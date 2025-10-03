package coordinator

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/baton/baton"
	"github.com/groundhogchain/chuckd/catchup"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/multiplex"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/repair"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/scp/scpd"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/chuckd/wallet"
	proto "github.com/groundhogchain/protocol-go"
)

type System struct {
	cfg proto.Config
	//
	inside  libnet.Network
	outside libnet.Network
	//
	Signer protoutil.Signatory
	//
	Coord     *coord.CoordinatorService
	Discovery *discovery.DiscoveryService
	Router    router.Router
	Dogwood   *dogwood.DogwoodService
	Multiplex *multiplex.Multiplex
	//
	Live    *LiveHandler
	Baton   *baton.Baton
	Catchup *catchup.CatchupService
	//
	Submitted *mempool.SubmittedMempool
	Committed *mempool.CommittedMempool
	Repair    *repair.RepairService
	//
	Submitter *Submitter
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
	callsPerExec int,
	batchSize int,
) *System {

	liblog.Infof("initializing validator %v", protoutil.VIDString(protoutil.PublicVID(cfg.Validator_config.Private_info.Public_info)))

	sys := &System{
		cfg:     cfg,
		inside:  inside,
		outside: outside,
		Signer:  protoutil.SignerFromPrivate(cfg.Validator_config.Private_info),
	}

	sys.Coord = coord.NewCoordinatorService(ctx, inside, runtime.NewRandFromRand(rand))

	sys.Discovery = discovery.NewDiscoveryService(
		ctx,
		outside,
		*cfg.Discovery_config,
		*cfg.Validator_config,
		*cfg.Scp_config,
	)

	sys.Router = router.NewDynamicUniformRouter(ctx, sys.Discovery, int(cfg.Router_config.Tree_fan_out))

	sys.Dogwood = dogwood.NewDogwoodService(
		ctx,
		*cfg.Dogwood_config,
		libtime.RealTime{},
		runtime.NewRandFromRand(rand),
		outside,
		inside,
		sys.Discovery,
		sys.Router,
	)

	sys.Multiplex = multiplex.NewMultiplex(ctx, sys.Dogwood)

	txTTL := time.Duration(cfg.Mempool_config.Transaction_ttl_msec) * time.Millisecond

	sys.Submitted = mempool.NewTransactionMempoolReal(ctx, txTTL)
	sys.Committed = mempool.NewCommittedMempoolReal(ctx, 2*txTTL)
	sys.Repair = repair.NewRepairService(ctx, sys.Submitted)

	sys.Live = NewLiveHandler(
		sys.Signer,
		libtime.RealTime{},
		inside,
		outside,
		sys.Coord,
		sys.Router,
		sys.Submitted,
		sys.Committed,
		sys.Dogwood,
		runtime.NewRandFromRand(rand),
	)

	sys.Baton = baton.NewBaton(ctx, *cfg.Baton_config, sys.Live, callsPerExec, batchSize)

	sys.Catchup = catchup.NewCatchupService(
		ctx,
		sys.Signer,
		libtime.RealTime{},
		outside,
		nil, // TODO: dialDb for deep archival access to catchup blocks, currently not used
		time.Duration(cfg.Catchup_config.Catchup_block_ttl_sec)*time.Second,
		cfg.Catchup_config.Catchup_block_age,
	)

	sys.Submitter = NewSubmitter(
		ctx,
		libtime.RealTime{},
		sys.Submitted,
		sys.Multiplex,
		int(cfg.Mempool_config.Submission_lot_size),
		time.Duration(cfg.Mempool_config.Submission_lot_wait_msec)*time.Millisecond,
	)

	sys.Wallet = wallet.NewWalletGateway(ctx, sys.Submitter, cfg.Wallet_config.Secret_key, cfg.Wallet_config.Users)

	sys.insideSrv = NewServer(ctx,
		proto.Coordinator1_Server{Srv: sys.Coord},
		proto.Wallet1_Server{Srv: sys.Wallet.Service()},
	)

	sys.outsideSrv = NewServer(ctx,
		proto.Discovery1_Server{Srv: sys.Discovery},
		proto.Dogwood1_Server{Srv: sys.Dogwood},
		proto.Catchup1_Server{Srv: sys.Catchup},
		proto.Repair1_Server{Srv: sys.Repair},
	)

	return sys
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
	InAddr         string
	OutAddr        string
	WalletAddr     string
	SCPDaemonPath  string
	SCPHandleAddr  string
	SCPControlAddr string
	IsValidator    bool
}

func (s *System) Start(ctx context.Context, opcfg OperationalConfig) {

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

	s.Dogwood.Start(ctx)                              // dogwood
	s.insideSrv.Start(ctx, s.inside, opcfg.InAddr)    // coord, wallet
	s.outsideSrv.Start(ctx, s.outside, opcfg.OutAddr) // discovery, dogwood, catchup, repair
	s.Wallet.StartHTTP(ctx, opcfg.WalletAddr)
	s.Baton.Start(ctx)

	// drive received envelopes into scp_control
	go func() {
		for {
			sender, env := s.Multiplex.ReceiveEnvelope(ctx)
			liblog.Debugf("received envelope from %v: %v", protoutil.VIDString(sender), protoutil.SCPEnvelopeShortString(*env))
			scpController.ReceivedEnvelope(ctx, *env)
		}
	}()

	// drive disseminated txs into mempool
	go func() {
		m := xdrutil.NewXdrMarshaler()
		for {
			sender, submittedTxs := s.Multiplex.ReceiveSubmittedTxs(ctx)
			liblog.Debugf("received %d txs from %v: %v",
				len(submittedTxs.Transactions),
				protoutil.VIDString(sender),
				form.Display(protoutil.HashSignedTxs(m, submittedTxs.Transactions)))
			s.Submitted.AddTxs(ctx, protoutil.Unpack(submittedTxs.Transactions)...)
		}
	}()
}

func (s *System) Close() {
	s.insideSrv.Close()
	s.outsideSrv.Close()
}
