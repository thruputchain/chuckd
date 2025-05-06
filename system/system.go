package system

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/net/broadcast"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/multiplex"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/net/spectra"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/wallet"
	proto "github.com/groundhogchain/protocol-go"
)

type System struct {
	Coord     *coord.CoordinatorService
	Discovery *discovery.DiscoveryService
	Router    router.Router
	Broadcast *broadcast.BroadcastService
	Spectra   *spectra.SpectraTransport
	Multiplex *multiplex.Multiplex // XXX: needs a driver for receiving!
	//
	Proposed    *mempool.ProposedMempool
	Transaction *mempool.TransactionMempool
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

	sys := &System{}

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

	sys.Multiplex = multiplex.NewMultiplex(sys.Spectra)

	// XXX: SCP solo follower
	// XXX: baton
	// XXX: set baton as scp handler

	sys.Proposed = mempool.NewProposedMempool(
		ctx,
		time.Duration(cfg.Mempool_config.Proposed_batch_ttl_msec)*time.Millisecond,
	)
	sys.Transaction = mempool.NewTransactionMempool(
		ctx,
		time.Duration(cfg.Mempool_config.Transaction_ttl_msec)*time.Millisecond,
	)

	// XXX: add wallet config to k8s
	sys.Wallet = wallet.NewWalletGateway(ctx, sys.Transaction, cfg.Wallet_config.Secret_key, cfg.Wallet_config.Users)

	// XXX: driver for multiplex

	sys.internal = NewServer(ctx,
		proto.Coordinator1_Server{Srv: sys.Coord},
		proto.Wallet1_Server{Srv: sys.Wallet.Service()},
	)

	sys.external = NewServer(ctx,
		proto.Discovery1_Server{Srv: sys.Discovery},
		proto.Broadcast1_Server{Srv: sys.Broadcast},
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
