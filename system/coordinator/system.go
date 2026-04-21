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

	s.Storage.Start(s.ctx, opcfg.StorageDir)

	// start internal rpcs before connecting to scheduler, so scheduler can connect back
	s.insideSrv.Start(s.ctx, s.inside, opcfg.InAddr) // serves: coord_sched.x, wallet.x

	schedConn, err := sched.ConnectToScheduler(s.ctx, s.inside, opcfg.SchedAddr, opcfg.InAddr)
	must.NoError(ctx, err)
	s.SchedConn = schedConn

	s.RepairService.SetSchedConn(schedConn)
	s.Live.SetScheduler(schedConn)

	s.UnvettedSubmitter.Start(s.ctx, schedConn)

	// handshake with scpd
	scpHandler, scpController, scpdKill := scpd.StartSCPDaemon(
		s.ctx,
		opcfg.SCPDaemonPath,
		opcfg.SCPControlAddr,
		opcfg.SCPHandleAddr,
		opcfg.IsValidator,
		*s.cfg.Scp_config,
		protoutil.ToSecretKey(s.cfg.Validator_config.Private_info.Secret_key),
	)
	s.scpHandler = scpHandler
	s.scpController = scpController
	s.scpdKill = scpdKill

	// broker must be ready for operation before the baton starts
	s.SCPBroker.SetSCPController(scpController)

	s.Discovery.Start(s.ctx, scpController)
	s.Live.SetSCP(scpController)
	scpHandler.SetHandlers(s.Multiplex, s.Discovery, s.Baton.SCPHandler())

	s.Dogwood.Start(s.ctx) // dogwood

	s.outsideSrv.Start(s.ctx, s.outside, opcfg.OutAddr) // discovery, dogwood, catchup, repair

	s.Wallet.StartHTTP(s.ctx, opcfg.WalletAddr)

	s.Dogwood.WaitReady(s.ctx) // wait until routes have been disseminated

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case <-s.ctx.Done():
			return
		case <-s.genesisReady: // wait for genesis to be in the db
		}
		liblog.Infof(ctx, "genesis ready")

		s.Baton.Start(s.ctx)

		// drive vetted transactions to dissemination
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			buf := NewBatchingBufferForParallelSignedTxs(
				s.ctx,
				s.time,
				int(s.cfg.Mempool_config.Submission_lot_size),
				time.Duration(s.cfg.Mempool_config.Submission_lot_wait_msec)*time.Millisecond,
				func(ctx context.Context, batch proto.ParallelSignedTransactionBatch) {
					liblog.Debugf(ctx, "disseminating %d vetted txs", xdrutil.SizeOfBatch(batch))
					s.Multiplex.SendSubmittedTxs(
						ctx,
						&proto.SubmittedTransactions{Transactions: batch},
					)
				},
			)
			for {
				batch, ok := s.CoordScheduler.ReceiveVetted(s.ctx)
				if !ok {
					return
				}
				liblog.Debugf(ctx, "received %d vetted txs from scheduler", xdrutil.SizeOfBatch(batch))
				buf.Submit(s.ctx, batch)
			}
		}()

		// drive received envelopes into scp_control
		s.RepairService.SetSCPBroker(s.SCPBroker)
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				_, env, ok := s.Multiplex.ReceiveEnvelope(s.ctx)
				if !ok {
					return
				}
				s.SCPBroker.ReceivedEnvelope(s.ctx, *env)
			}
		}()

		// drive received txs into mempool
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			m := libxdr.NewXdrMarshaler()
			_ = m
			for {
				sender, submittedTxs, ok := s.Multiplex.ReceiveSubmittedTxs(s.ctx)
				if !ok {
					return
				}
				_ = sender

				// liblog.Debugf("received %d txs from %v: %v",
				// 	len(submittedTxs.Transactions),
				// 	protoutil.VIDString(sender),
				// 	form.Display(protoutil.HashSignedTxs(m, submittedTxs.Transactions)))

				err := schedConn.Schedulercoord_submit_vetted(
					s.ctx,
					proto.ValidatorID{}, // XXX: who vetted them is not known
					submittedTxs.Transactions)
				must.NoError(ctx, err)
			}
		}()

		// drive received consensus values into cvpool
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				sender, cv, ok := s.Multiplex.ReceiveConsensusValue(s.ctx)
				if !ok {
					return
				}
				_ = sender
				if !protoutil.IsSaneConsensusValue(cv) {
					liblog.Errorf(ctx, "received cv at height %d is not sane, dropping", cv.Nomination.Block_height)
					continue
				}
				s.CVPool.CVPoolAdd(s.ctx, *cv, protoutil.HashCVFast(cv))
			}
		}()
	}()

	return Control{
		system:       s,
		conn:         schedConn,
		genesisReady: s.genesisReady,
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

func (s *System) signalGenesisIsReady() {
	s.genesisOnce.Do(func() {
		close(s.genesisReady)
	})
}

func (s *System) Stop() {
	s.stopOnce.Do(func() {
		s.signalGenesisIsReady()
		if s.cancel != nil {
			s.cancel()
		}
		if s.outsideSrv != nil {
			liblog.Infof(s.ctx, "coordinator stopping outside network")
			s.outsideSrv.Stop()
		}
		if s.UnvettedSubmitter != nil {
			s.UnvettedSubmitter.Stop()
		}
		if s.CoordScheduler != nil {
			s.CoordScheduler.Stop()
		}
		if s.insideSrv != nil {
			liblog.Infof(s.ctx, "coordinator stopping inside network")
			s.insideSrv.Stop()
		}
		if s.Wallet != nil {
			liblog.Infof(s.ctx, "coordinator stopping wallet gateway")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			s.Wallet.Stop(ctx)
			cancel()
		}
		if s.scpController != nil {
			s.scpController.Close()
		}
		if s.scpHandler != nil {
			s.scpHandler.Stop()
		}
		if s.scpdKill != nil {
			liblog.Infof(s.ctx, "coordinator killing scpd")
			s.scpdKill()
		}
		if s.SchedConn != nil {
			liblog.Infof(s.ctx, "coordinator closing scheduler connection")
			s.SchedConn.Close()
		}
		s.wg.Wait()
		liblog.Infof(s.ctx, "coordinator stopped")
	})
}

func (s *System) Close() {
	s.Stop()
}
