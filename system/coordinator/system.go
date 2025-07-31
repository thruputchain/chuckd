package coordinator

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/baton/baton"
	"github.com/groundhogchain/chuckd/catchup"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/multiplex"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/scp/solo/follower"
	"github.com/groundhogchain/chuckd/util/protoutil"
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
	Live     *LiveHandler
	Baton    *baton.Baton
	Follower *follower.SoloFollowerService
	Catchup  *catchup.CatchupService
	//
	Submitted *mempool.SubmittedMempool
	Committed *mempool.CommittedMempool
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

	sys := &System{
		cfg:     cfg,
		inside:  inside,
		outside: outside,
		Signer:  protoutil.SignerFromPrivate(cfg.Validator_config.Private_info),
	}

	sys.Coord = coord.NewCoordinatorService(ctx, inside, runtime.NewRandFromRand(rand))

	sys.Discovery = discovery.NewDiscoveryService(
		ctx,
		*cfg.Discovery_config,
		outside,
		*cfg.Validator_config,
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

	sys.Baton = baton.NewBaton(ctx, sys.Live, callsPerExec, batchSize)
	sys.Follower = follower.NewSoloFollowerService(
		ctx,
		outside,
		cfg.Solo_config.Leader,
		cfg.Validator_config.Private_info,
		sys.Baton.SCPHandler(),
	)
	sys.Live.SetSCP(sys.Follower)

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

	// drive disseminated txs into mempool
	go func() {
		for {
			_, submittedTxs := sys.Multiplex.ReceiveSubmittedTxs(ctx)
			liblog.Debugf("received %d txs", len(submittedTxs.Transactions))
			sys.Submitted.AddTxs(ctx, protoutil.Unpack(submittedTxs.Transactions)...)
		}
	}()

	sys.Wallet = wallet.NewWalletGateway(ctx, sys.Submitter, cfg.Wallet_config.Secret_key, cfg.Wallet_config.Users)

	sys.insideSrv = NewServer(ctx,
		proto.Coordinator1_Server{Srv: sys.Coord},
		proto.Wallet1_Server{Srv: sys.Wallet.Service()},
	)

	sys.outsideSrv = NewServer(ctx,
		proto.Discovery1_Server{Srv: sys.Discovery},
		proto.Dogwood1_Server{Srv: sys.Dogwood},
		proto.Catchup1_Server{Srv: sys.Catchup},
		proto.SoloFollower1_Server{Srv: sys.Follower},
	)

	return sys
}

func (s *System) Start(ctx context.Context, inAddr string, outAddr string, walletAddr string) {
	s.Dogwood.Start(ctx)
	s.insideSrv.Start(ctx, s.inside, inAddr)
	s.outsideSrv.Start(ctx, s.outside, outAddr)
	s.Wallet.StartHTTP(ctx, walletAddr)
	s.Baton.Start(ctx, time.Duration(s.cfg.Baton_config.Heartbeat_msec)*time.Millisecond)
}

func (s *System) Close() {
	s.insideSrv.Close()
	s.outsideSrv.Close()
}
