package coordinator

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/multiplex"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/baton/baton"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/catchup"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/repair/repairagent"
	"github.com/groundhogchain/chuckd/sub/repair/repairservice"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/sub/scp/scpbroker"
	"github.com/groundhogchain/chuckd/sub/scp/scpd"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/sub/wallet"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libtime"
	proto "github.com/groundhogchain/protocol-go"
)

type System struct {
	cfg proto.Config
	//
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	//
	time    libtime.Time
	inside  libnet.Network
	outside libnet.Network
	//
	signer protoutil.Signatory
	//
	CoordScheduler *sched.CoordSchedulerService
	//
	Discovery *discovery.DiscoveryService
	Router    router.Router
	Dogwood   *dogwood.DogwoodService
	Multiplex *multiplex.Multiplex
	//
	Storage   *storage.DiskStorage
	SCPBroker *scpbroker.SCPBroker
	Live      *LiveHandler
	Baton     *baton.Baton
	CBPool    *catchup.CatchupBlockPool
	Catchup   *catchup.CatchupService
	//
	CVPool        *cvpool.ConsensusValuePool
	RepairService *repairservice.RepairService
	RepairAgent   *repairagent.RepairAgent
	//
	UnvettedSubmitter *UnvettedSubmitter
	//
	Wallet *wallet.WalletGateway
	//
	insideSrv  *Server
	outsideSrv *Server
	//
	SchedConn     sched.SchedulerCoordConn
	scpHandler    *scpd.SCPHandler
	scpController *scpd.SCPController
	scpdKill      func()
	//
	genesisReady chan struct{}
	genesisOnce  sync.Once
	stopOnce     sync.Once
}

func NewSystem(
	ctx context.Context,
	notify Notify,
	rand *rand.Rand,
	cfg proto.Config,
	inside libnet.Network,
	outside libnet.Network,
) *System {
	sysCtx, sysCancel := context.WithCancel(ctx)

	vid := protoutil.PublicVID(cfg.Validator_config.Private_info.Public_info)
	liblog.Infof(ctx, "initializing validator %v", protoutil.VIDString(vid))

	if notify == nil {
		notify = IgnoreNotify{}
	}
	s := &System{
		cfg:          cfg,
		ctx:          sysCtx,
		cancel:       sysCancel,
		time:         libtime.RealTime{},
		inside:       inside,
		outside:      outside,
		signer:       protoutil.SignerFromPrivate(cfg.Validator_config.Private_info),
		genesisReady: make(chan struct{}),
	}

	s.CoordScheduler = sched.NewCoordSchedulerService()

	s.Discovery = discovery.NewDiscoveryService(
		s.ctx,
		outside,
		*cfg.Discovery_config,
		*cfg.Validator_config,
		*cfg.Scp_config,
	)

	s.Router = router.NewDynamicUniformRouter(s.ctx, s.Discovery, int(cfg.Router_config.Tree_fan_out))

	s.Dogwood = dogwood.NewDogwoodService(
		s.ctx,
		*cfg.Dogwood_config,
		s.time,
		runtime.NewRandFromRand(rand),
		outside,
		inside,
		s.Discovery,
		s.Router,
	)

	s.Storage = storage.NewStorage(s.ctx, storage.Config{
		Freq:             time.Duration(cfg.Storage_config.Reclaim_interval_msec) * time.Millisecond,
		MaxCatchupBlocks: int(cfg.Storage_config.Max_catchup_blocks),
		MaxSCPSlots:      int(cfg.Storage_config.Max_scp_slots),
	})

	s.Multiplex = multiplex.NewMultiplex(s.ctx, vid, s.Dogwood, s.Storage)

	s.CVPool = cvpool.NewCVPool(s.ctx, s.Storage)
	s.CBPool = catchup.NewCatchupBlockPool(s.ctx, s.Storage)

	s.RepairService = repairservice.NewRepairService(s.CVPool, s.CBPool)
	s.RepairAgent = repairagent.NewRepairAgent(
		*cfg.Repair_config,
		outside,
		runtime.NewRandFromRand(rand),
		s.signer,
		s.Router,
		s.CVPool,
	)
	s.SCPBroker = scpbroker.NewSCPBroker(s.ctx, s.Router, s.Storage, s.RepairAgent)

	s.Live = NewLiveHandler(
		notify,
		s.signer,
		libtime.RealTime{},
		inside,
		outside,
		s.Router,
		s.RepairAgent,
		s.RepairService,
		s.CVPool,
		s.CBPool,
		s.Multiplex,
		s.Storage,
		s.SCPBroker,
		runtime.NewRandFromRand(rand),
	)

	s.Baton = baton.NewBaton(s.ctx, *cfg.Baton_config, s.Live)

	s.Catchup = catchup.NewCatchupService(
		s.ctx,
		s.signer,
		libtime.RealTime{},
		outside,
		s.CBPool,
		int(cfg.Storage_config.Max_catchup_blocks),
	)

	s.UnvettedSubmitter = NewUnvettedSubmitter(
		s.ctx,
		libtime.RealTime{},
		int(cfg.Mempool_config.Submission_lot_size),
		time.Duration(cfg.Mempool_config.Submission_lot_wait_msec)*time.Millisecond,
	)

	s.Wallet = wallet.NewWalletGateway(
		s.ctx,
		s.UnvettedSubmitter,
		cfg.Wallet_config.Secret_key,
		cfg.Wallet_config.Users,
	)

	s.insideSrv = NewServer(s.ctx,
		proto.CoordScheduler1_Server{Srv: s.CoordScheduler},
		proto.Wallet1_Server{Srv: s.Wallet.Service()},
	)

	s.outsideSrv = NewServer(s.ctx,
		proto.Discovery1_Server{Srv: s.Discovery},
		proto.Dogwood1_Server{Srv: s.Dogwood},
		proto.Catchup1_Server{Srv: s.Catchup},
		proto.Repair1_Server{Srv: s.RepairService},
	)

	return s
}

func (s *System) Config() proto.Config {
	return s.cfg
}

func (s *System) SetLevers(levers batonkit.BatonLevers) {
	s.Baton.SetLevers(levers)
}

func (s *System) GetLevers() batonkit.BatonLevers {
	return s.Baton.GetLevers()
}

func (s *System) SetHooks(hooks batonkit.BatonHooks) {
	s.Baton.SetHooks(hooks)
}

func (s *System) PauseNominating() {
	s.Baton.PauseNominating()
}

func (s *System) ResumeNominating() {
	s.Baton.ResumeNominating()
}

func (s *System) signalGenesisIsReady() {
	s.genesisOnce.Do(func() {
		close(s.genesisReady)
	})
}
