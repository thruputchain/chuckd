package coordinator

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/baton/baton"
	"github.com/groundhogchain/chuckd/catchup"
	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/multiplex"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/repair"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sched"
	"github.com/groundhogchain/chuckd/scp/scpd"
	"github.com/groundhogchain/chuckd/storage"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/chuckd/wallet"
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
	Storage *storage.DiskStorage
	Live    *LiveHandler
	Baton   *baton.Baton
	Catchup *catchup.CatchupService
	//
	RepairService *repair.RepairService
	RepairAgent   *repair.RepairAgent
	//
	WalletSubmitter *WalletSubmitter
	//
	Wallet *wallet.WalletGateway
	//
	insideSrv  *Server
	outsideSrv *Server
}

func NewSystem(
	ctx context.Context,
	rand *rand.Rand,
	cfg proto.Config,
	inside libnet.Network,
	outside libnet.Network,
) *System {

	vid := protoutil.PublicVID(cfg.Validator_config.Private_info.Public_info)
	liblog.Infof("initializing validator %v", protoutil.VIDString(vid))

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

	s.RepairService = repair.NewRepairService()
	s.RepairAgent = repair.NewRepairAgent(
		*cfg.Repair_config,
		outside,
		s.signer,
	)

	s.Storage = storage.NewStorage(ctx, storage.Config{
		Freq:             time.Duration(cfg.Storage_config.Reclaim_interval_msec) * time.Millisecond,
		MaxCatchupBlocks: int(cfg.Storage_config.Max_catchup_blocks),
		MaxSCPSlots:      int(cfg.Storage_config.Max_scp_slots),
	})
	s.Live = NewLiveHandler(
		s.signer,
		libtime.RealTime{},
		inside,
		outside,
		s.Router,
		s.RepairAgent,
		s.Dogwood,
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

	s.WalletSubmitter = NewWalletSubmitter(
		ctx,
		libtime.RealTime{},
		int(cfg.Mempool_config.Submission_lot_size),
		time.Duration(cfg.Mempool_config.Submission_lot_wait_msec)*time.Millisecond,
	)

	s.Wallet = wallet.NewWalletGateway(
		ctx,
		s.WalletSubmitter,
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
	InAddr         string
	OutAddr        string
	WalletAddr     string
	SCPDaemonPath  string
	SCPHandleAddr  string
	SCPControlAddr string
	IsValidator    bool
}

func (s *System) Start(ctx context.Context, opcfg OperationalConfig) {

	s.Storage.Start(ctx, opcfg.StorageDir)

	// intro to scheduler
	schedConn, err := sched.DialSchedulerCoordClient(ctx, s.inside, opcfg.SchedAddr)
	must.NoError(ctx, err)

	err = sched.Intro(ctx, schedConn, opcfg.InAddr)
	must.NoError(ctx, err)

	s.RepairService.SetSchedConn(schedConn)
	s.Live.SetScheduler(schedConn)

	s.WalletSubmitter.Start(ctx, schedConn)

	// handshake with scpd
	scpHandler, scpController := scpd.StartSCPDaemon(
		ctx,
		opcfg.SCPDaemonPath,
		opcfg.SCPControlAddr,
		opcfg.SCPHandleAddr,
		opcfg.IsValidator,
		*s.cfg.Scp_config,
		protoutil.ToSecretKey(s.cfg.Validator_config.Private_info.Secret_key),
	)
	s.Live.SetSCP(scpController)
	scpHandler.SetHandlers(s.Multiplex, s.Discovery, s.Baton.SCPHandler())

	s.Dogwood.Start(ctx) // dogwood

	s.insideSrv.Start(ctx, s.inside, opcfg.InAddr)    // coord, wallet
	s.outsideSrv.Start(ctx, s.outside, opcfg.OutAddr) // discovery, dogwood, catchup, repair

	s.Wallet.StartHTTP(ctx, opcfg.WalletAddr)

	s.Dogwood.WaitReady(ctx) // wait until routes have been disseminated

	s.Baton.Start(ctx)

	// drive vetted transactions to dissemination
	go func() {
		buf := libflow.NewBuffer(
			ctx,
			s.time,
			func(ctx context.Context, txs []proto.SignedTransaction) {
				liblog.Debugf("disseminating %d vetted txs", len(txs))
				s.Multiplex.Disseminate(ctx, &proto.SubmittedTransactions{Transactions: txs})
			},
			int(s.cfg.Mempool_config.Submission_lot_size),
			time.Duration(s.cfg.Mempool_config.Submission_lot_wait_msec)*time.Millisecond,
		)
		for {
			txs := s.CoordScheduler.ReceiveVetted()
			liblog.Debugf("received %d vetted txs from scheduler", len(txs))
			buf.Submit(ctx, txs...)
		}
	}()

	// drive received envelopes into scp_control
	go func() {
		for {
			_, env := s.Multiplex.ReceiveEnvelope(ctx)
			scpController.ReceivedEnvelope(ctx, *env)
		}
	}()

	// drive disseminated txs into mempool
	go func() {
		m := xdrutil.NewXdrMarshaler()
		_ = m
		for {
			sender, submittedTxs := s.Multiplex.ReceiveSubmittedTxs(ctx)
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
}

func (s *System) Close() {
	s.insideSrv.Close()
	s.outsideSrv.Close()
}
