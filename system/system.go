package system

import (
	"context"
	"math/rand"

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
	treeFanOut int, // fanout of multicast trees
) *System {

	sys := &System{}

	sys.Coord = coord.NewCoordinatorService(ctx, libnet.TCPNet{}, runtime.NewRandFromRand(rr))

	sys.Discovery = discovery.NewDiscoveryService(
		ctx,
		discovery.DefaultDiscoveryConfig, // XXX: convert to proto
		libnet.TCPNet{},
		*cfg.Validator_config,
	)

	sys.Router = router.NewBasicRouter(ctx, sys.Discovery, treeFanOut)

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

	sys.Proposed = mempool.NewProposedMempool(ctx, batchTTLXXX)
	sys.Transaction = mempool.NewTransactionMempool(ctx, txTTLXXX)

	sys.Wallet = wallet.NewWalletGateway(ctx, sys.Transaction, jwtKeyXXX, usersXXX)

	// XXX: driver for multiplex

	sys.internal = NewServer(ctx,
		proto.Coordinator1_Server{Srv: sys.Coord},
		proto.Wallet1_Server{Srv: sys.Wallet.Service()},
	)

	sys.external = NewServer(ctx,
		proto.Discovery1_Server{Srv: sys.Discovery},
		proto.Broadcast1_Server{Srv: sys.Broadcast},
	)

	sys.Wallet.StartHTTP(ctx, walletHTTPAddrXXX)

	return sys
}

func (s *System) Start(ctx context.Context, inaddr string, exaddr string) {
	s.internal.Start(ctx, inaddr)
	s.external.Start(ctx, exaddr)
}

func (s *System) Close() {
	s.internal.Close()
	s.external.Close()
}
