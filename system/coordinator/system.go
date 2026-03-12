package coordinator

import (
	"context"
	"math/rand"
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
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type System struct {
	cfg proto.Config
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
	SchedConn sched.SchedulerCoordConn
	scpdKill  func()
}

func NewSystem(
	ctx context.Context,
	notify Notify,
	rand *rand.Rand,
	cfg proto.Config,
	inside libnet.Network,
	outside libnet.Network,
) *System {

	vid := protoutil.PublicVID(cfg.Validator_config.Private_info.Public_info)
	liblog.Infof("initializing validator %v", protoutil.VIDString(vid))

	if notify == nil {
		notify = IgnoreNotify{}
	}
	s := &System{
		cfg:     cfg,
		time:    libtime.RealTime{},
		inside:  inside,
		outside: outside,
		signer:  protoutil.SignerFromPrivate(cfg.Validator_config.Private_info),
	}

	s.CoordScheduler = sched.NewCoordSchedulerService()

	s.Discovery = discovery.NewDiscoveryService(
		ctx,
		outside,
		*cfg.Discovery_config,
		*cfg.Validator_config,
		*cfg.Scp_config,
	)

	s.Router = router.NewDynamicUniformRouter(ctx, s.Discovery, int(cfg.Router_config.Tree_fan_out))

	s.Dogwood = dogwood.NewDogwoodService(
		ctx,
		*cfg.Dogwood_config,
		s.time,
		runtime.NewRandFromRand(rand),
		outside,
		inside,
		s.Discovery,
		s.Router,
	)

	s.Multiplex = multiplex.NewMultiplex(ctx, vid, s.Dogwood)

	s.Storage = storage.NewStorage(ctx, storage.Config{
		Freq:             time.Duration(cfg.Storage_config.Reclaim_interval_msec) * time.Millisecond,
		MaxCatchupBlocks: int(cfg.Storage_config.Max_catchup_blocks),
		MaxSCPSlots:      int(cfg.Storage_config.Max_scp_slots),
	})
	s.CVPool = cvpool.NewConsensusValuePool(ctx, s.Storage)
	s.CBPool = catchup.NewCatchupBlockPool(ctx, s.Storage)

	s.RepairService = repairservice.NewRepairService(s.CVPool)
	s.RepairAgent = repairagent.NewRepairAgent(
		*cfg.Repair_config,
		outside,
		runtime.NewRandFromRand(rand),
		s.signer,
		s.Router,
		s.CVPool,
	)
	s.SCPBroker = scpbroker.NewSCPBroker(ctx, s.Router, s.Storage, s.RepairAgent)

	s.Live = NewLiveHandler(
		notify,
		s.signer,
		libtime.RealTime{},
		inside,
		outside,
		s.Router,
		s.RepairAgent,
		s.CVPool,
		s.CBPool,
		s.Multiplex,
		s.Storage,
		s.SCPBroker,
		runtime.NewRandFromRand(rand),
	)

	s.Baton = baton.NewBaton(ctx, *cfg.Baton_config, s.Live)

	s.Catchup = catchup.NewCatchupService(
		ctx,
		s.signer,
		libtime.RealTime{},
		outside,
		time.Duration(cfg.Catchup_config.Catchup_block_ttl_sec)*time.Second,
		cfg.Catchup_config.Catchup_block_age,
	)

	s.UnvettedSubmitter = NewUnvettedSubmitter(
		ctx,
		libtime.RealTime{},
		int(cfg.Mempool_config.Submission_lot_size),
		time.Duration(cfg.Mempool_config.Submission_lot_wait_msec)*time.Millisecond,
	)

	s.Wallet = wallet.NewWalletGateway(
		ctx,
		s.UnvettedSubmitter,
		cfg.Wallet_config.Secret_key,
		cfg.Wallet_config.Users,
	)

	s.insideSrv = NewServer(ctx,
		proto.CoordScheduler1_Server{Srv: s.CoordScheduler},
		proto.Wallet1_Server{Srv: s.Wallet.Service()},
	)

	s.outsideSrv = NewServer(ctx,
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

// System dependencies:
//	[] means delayed set after init
//
//	 coord
//
//	 discovery
//	 router --> discovery
//	 dogwood --> discovery, router
//
//	 catchup --> db
//
//	 multiplex --> dogwood
//	 submitter --> multiplex
//
//	 live --> [ scp_control ], coord (db)
//	 baton --> live
//	 [ scp_handler --> baton ]
//
//	 [ multiplex_receiver --> submitter, scp_control ]
//
//	 wallet --> submitter
//
// Delayed connections:
//
//	 live --> scp_control
//	 multiplex_receiver --> scp_control
//	 scp_handler --> multiplex, discovery, baton
//
//	 multiplex_receiver --> submitter, scp_control

type OperationalConfig struct {
	StorageDir     string
	SchedAddr      string
	DebugPort      int // used by coordinator bench only
	InAddr         string
	OutAddr        string
	WalletAddr     string
	SCPDaemonPath  string
	SCPHandleAddr  string
	SCPControlAddr string
	IsValidator    bool
}

func (s *System) Start(ctx context.Context, opcfg OperationalConfig) Control {

	s.Storage.Start(ctx, opcfg.StorageDir)

	// start internal rpcs before connecting to scheduler, so scheduler can connect back
	s.insideSrv.Start(ctx, s.inside, opcfg.InAddr) // serves: coord_sched.x, wallet.x

	schedConn, err := sched.ConnectToScheduler(ctx, s.inside, opcfg.SchedAddr, opcfg.InAddr)
	must.NoError(ctx, err)
	s.SchedConn = schedConn

	s.RepairService.SetSchedConn(schedConn)
	s.Live.SetScheduler(schedConn)

	s.UnvettedSubmitter.Start(ctx, schedConn)

	// handshake with scpd
	scpHandler, scpController, scpdKill := scpd.StartSCPDaemon(
		ctx,
		opcfg.SCPDaemonPath,
		opcfg.SCPControlAddr,
		opcfg.SCPHandleAddr,
		opcfg.IsValidator,
		*s.cfg.Scp_config,
		protoutil.ToSecretKey(s.cfg.Validator_config.Private_info.Secret_key),
	)
	s.scpdKill = scpdKill

	s.Discovery.Start(ctx, scpController)
	s.Live.SetSCP(scpController)
	scpHandler.SetHandlers(s.Multiplex, s.Discovery, s.Baton.SCPHandler())

	s.Dogwood.Start(ctx) // dogwood

	s.outsideSrv.Start(ctx, s.outside, opcfg.OutAddr) // discovery, dogwood, catchup, repair

	s.Wallet.StartHTTP(ctx, opcfg.WalletAddr)

	s.Dogwood.WaitReady(ctx) // wait until routes have been disseminated

	genesisReady := make(chan struct{})
	go func() {
		<-genesisReady // wait for genesis to be in the db
		liblog.Infof("genesis ready")

		s.Baton.Start(ctx)

		// drive vetted transactions to dissemination
		go func() {
			buf := NewBatchingBufferForParallelSignedTxs(
				ctx,
				s.time,
				int(s.cfg.Mempool_config.Submission_lot_size),
				time.Duration(s.cfg.Mempool_config.Submission_lot_wait_msec)*time.Millisecond,
				func(ctx context.Context, batch proto.ParallelSignedTransactionBatch) {
					liblog.Debugf("disseminating %d vetted txs", xdrutil.SizeOfBatch(batch))
					s.Multiplex.SendSubmittedTxs(
						ctx,
						&proto.SubmittedTransactions{Transactions: batch},
					)
				},
			)
			for {
				batch := s.CoordScheduler.ReceiveVetted()
				liblog.Debugf("received %d vetted txs from scheduler", xdrutil.SizeOfBatch(batch))
				buf.Submit(ctx, batch)
			}
		}()

		// drive received envelopes into scp_control
		s.SCPBroker.SetSCPController(scpController)
		s.RepairService.SetSCPBroker(s.SCPBroker)
		go func() {
			for {
				_, env, ok := s.Multiplex.ReceiveEnvelope(ctx)
				if !ok {
					return
				}
				s.SCPBroker.ReceivedEnvelope(ctx, *env)
			}
		}()

		// drive received txs into mempool
		go func() {
			m := libxdr.NewXdrMarshaler()
			_ = m
			for {
				sender, submittedTxs, ok := s.Multiplex.ReceiveSubmittedTxs(ctx)
				if !ok {
					return
				}
				_ = sender

				// liblog.Debugf("received %d txs from %v: %v",
				// 	len(submittedTxs.Transactions),
				// 	protoutil.VIDString(sender),
				// 	form.Display(protoutil.HashSignedTxs(m, submittedTxs.Transactions)))

				err := schedConn.Schedulercoord_submit_vetted(
					ctx,
					proto.ValidatorID{}, // XXX: who vetted them is not known
					submittedTxs.Transactions)
				must.NoError(ctx, err)
			}
		}()

		// drive received consensus values into cvpool
		go func() {
			for {
				sender, cv, ok := s.Multiplex.ReceiveConsensusValue(ctx)
				if !ok {
					return
				}
				_ = sender
				if !protoutil.IsSaneConsensusValue(cv) {
					liblog.Errorf("received cv at height %d is not sane, dropping", cv.Nomination.Block_height)
					continue
				}
				s.CVPool.ConsensusValuePoolAdd(ctx, *cv, protoutil.HashCVFast(cv))
			}
		}()
	}()

	return Control{
		system:       s,
		conn:         schedConn,
		genesisReady: genesisReady,
	}
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

func (s *System) Close() {
	liblog.Infof("coordinator stopping inside network")
	s.insideSrv.Close()
	liblog.Infof("coordinator stopping outside network")
	s.outsideSrv.Close()
	if s.SchedConn != nil {
		liblog.Infof("coordinator closing scheduler connection")
		s.SchedConn.Close()
	}
	liblog.Infof("coordinator killing scpd")
	s.scpdKill()
	liblog.Infof("coordinator stopped")
}
