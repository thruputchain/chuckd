package system

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/baton"
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
	Signer protoutil.Signer
	//
	Coord     *coord.CoordinatorService
	Discovery *discovery.DiscoveryService
	Router    router.Router
	Broadcast *broadcast.BroadcastService
	Spectra   *spectra.SpectraTransport
	Multiplex *multiplex.Multiplex
	//
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
	internal *Server
	external *Server
}

func NewSystem(
	ctx context.Context,
	rr *rand.Rand,
	cfg proto.Config,
) *System {

	sys := &System{
		Signer: protoutil.SignerFromPrivate(cfg.Validator_config.Private_info),
	}

	sys.Coord = coord.NewCoordinatorService(ctx, libnet.TCPNet{}, runtime.NewRandFromRand(rr))

	sys.Discovery = discovery.NewDiscoveryService(
		ctx,
		*cfg.Discovery_config,
		libnet.TCPNet{},
		*cfg.Validator_config,
	)

	sys.Router = router.NewBasicRouter(ctx, sys.Discovery, int(cfg.Router_config.Tree_fan_out))

	sys.Broadcast = broadcast.NewBroadcastService(
		ctx,
		*cfg.Broadcast_config,
		libnet.TCPNet{},
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

	sys.Baton = baton.NewBaton(ctx, hXXX)

	sys.Follower = follower.NewSoloFollowerService(
		ctx,
		libnet.TCPNet{},
		leaderXXX,
		cfg.Validator_config.Private_info,
		sys.Baton.SCPHandler(),
	)

	// XXX: set baton as scp handler

	sys.Catchup = catchup.NewCatchupService(
		ctx,
		sys.Signer,
		libtime.RealTime{},
		libnet.TCPNet{},
		nil, // XXX: currently not used
		ttl, //XXX
	)

	batchTTL := time.Duration(cfg.Mempool_config.Proposed_batch_ttl_msec) * time.Millisecond
	txTTL := time.Duration(cfg.Mempool_config.Transaction_ttl_msec) * time.Millisecond

	sys.Proposed = mempool.NewBatchMempoolReal(ctx, batchTTL)
	sys.Transaction = mempool.NewTransactionMempoolReal(ctx, txTTL)
	sys.Committed = mempool.NewCommittedMempoolReal(ctx, 2*max(batchTTL, txTTL))

	// XXX: add wallet config to k8s
	sys.Wallet = wallet.NewWalletGateway(ctx, sys.Transaction, cfg.Wallet_config.Secret_key, cfg.Wallet_config.Users)

	// drive disseminated batches into mempool
	go func() {
		for {
			_, batchMsg := sys.Multiplex.ReceiveBatch(ctx)
			sys.Proposed.AddBatch(ctx, batchMsg.Batch_transactions)
		}
	}()

	sys.internal = NewServer(ctx,
		proto.Coordinator1_Server{Srv: sys.Coord},
		proto.Wallet1_Server{Srv: sys.Wallet.Service()},
	)

	sys.external = NewServer(ctx,
		proto.Discovery1_Server{Srv: sys.Discovery},
		proto.Broadcast1_Server{Srv: sys.Broadcast},
		proto.Catchup1_Server{Srv: sys.Catchup},
	)

	return sys
}

func (s *System) Start(ctx context.Context, inAddr string, exAddr string, walletAddr string) {
	s.internal.Start(ctx, inAddr)
	s.external.Start(ctx, exAddr)
	s.Wallet.StartHTTP(ctx, walletAddr)
}

func (s *System) Close() {
	s.internal.Close()
	s.external.Close()
}
