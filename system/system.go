package system

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/baton/baton"
	"github.com/groundhogchain/chuckd/catchup"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/net/broadcast"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/multiplex"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/net/spectra"
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
	Broadcast *broadcast.BroadcastService
	Spectra   *spectra.SpectraTransport
	Multiplex *multiplex.Multiplex
	//
	Live     *LiveHandler
	Baton    *baton.Baton
	Follower *follower.SoloFollowerService
	Catchup  *catchup.CatchupService
	//
	Proposed    *mempool.BatchMempool
	Transaction *mempool.TransactionMempool
	Committed   *mempool.CommittedMempool
	//
	Wallet *wallet.WalletGateway
	//
	insideSrv  *Server
	outsideSrv *Server
}

func NewSystem(
	ctx context.Context,
	rr *rand.Rand,
	cfg proto.Config,
	inside libnet.Network,
	outside libnet.Network,
) *System {

	sys := &System{
		cfg:     cfg,
		inside:  inside,
		outside: outside,
		Signer:  protoutil.SignerFromPrivate(cfg.Validator_config.Private_info),
	}

	sys.Coord = coord.NewCoordinatorService(ctx, inside, runtime.NewRandFromRand(rr))

	sys.Discovery = discovery.NewDiscoveryService(
		ctx,
		*cfg.Discovery_config,
		outside,
		*cfg.Validator_config,
	)

	sys.Router = router.NewBasicRouter(ctx, sys.Discovery, int(cfg.Router_config.Tree_fan_out))

	sys.Broadcast = broadcast.NewBroadcastService(
		ctx,
		*cfg.Broadcast_config,
		outside,
		runtime.NewRandFromRand(rr),
		cfg.Validator_config.Private_info,
		sys.Router,
		broadcast.AcceptEverything{},
	)

	sys.Spectra = spectra.NewSpectraTransport(
		ctx,
		*cfg.Spectra_config,
		runtime.NewRandFromRand(rr),
		sys.Broadcast.BulkAPI(),
	)

	sys.Multiplex = multiplex.NewMultiplex(ctx, sys.Spectra)

	batchTTL := time.Duration(cfg.Mempool_config.Proposed_batch_ttl_msec) * time.Millisecond
	txTTL := time.Duration(cfg.Mempool_config.Transaction_ttl_msec) * time.Millisecond

	sys.Proposed = mempool.NewBatchMempoolReal(ctx, batchTTL)
	sys.Transaction = mempool.NewTransactionMempoolReal(ctx, txTTL)
	sys.Committed = mempool.NewCommittedMempoolReal(ctx, 2*max(batchTTL, txTTL))

	sys.Live = NewLiveHandler(
		sys.Signer,
		libtime.RealTime{},
		inside,
		outside,
		sys.Coord,
		sys.Router,
		sys.Proposed,
		sys.Transaction,
		sys.Committed,
		sys.Multiplex,
		runtime.NewRandFromRand(rr),
	)

	sys.Baton = baton.NewBaton(ctx, sys.Live)
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

	sys.Wallet = wallet.NewWalletGateway(ctx, sys.Transaction, cfg.Wallet_config.Secret_key, cfg.Wallet_config.Users)

	// drive disseminated batches into mempool
	go func() {
		for {
			_, batchMsg := sys.Multiplex.ReceiveBatch(ctx)
			sys.Proposed.AddBatch(ctx, batchMsg.Batch_transactions)
		}
	}()

	sys.insideSrv = NewServer(ctx,
		proto.Coordinator1_Server{Srv: sys.Coord},
		proto.Wallet1_Server{Srv: sys.Wallet.Service()},
	)

	sys.outsideSrv = NewServer(ctx,
		proto.Discovery1_Server{Srv: sys.Discovery},
		proto.Broadcast1_Server{Srv: sys.Broadcast},
		proto.Catchup1_Server{Srv: sys.Catchup},
		proto.SoloFollower1_Server{Srv: sys.Follower},
	)

	return sys
}

func (s *System) Start(ctx context.Context, inAddr string, outAddr string, walletAddr string) {
	s.insideSrv.Start(ctx, s.inside, inAddr)
	s.outsideSrv.Start(ctx, s.outside, outAddr)
	s.Wallet.StartHTTP(ctx, walletAddr)
	s.Baton.Start(ctx, time.Duration(s.cfg.Baton_config.Heartbeat_msec)*time.Millisecond)
}

func (s *System) Close() {
	s.insideSrv.Close()
	s.outsideSrv.Close()
}
