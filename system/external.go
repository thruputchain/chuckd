// Package server implements a server that hosts all external APIs on the same address.
package system

import (
	"context"
	"math/rand"
	"net"

	"github.com/groundhogchain/chuckd/broadcast"
	"github.com/groundhogchain/chuckd/discovery"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type ExternalServer struct {
	l         net.Listener
	discovery *discovery.DiscoveryService
	broadcast *broadcast.BroadcastService
}

func NewExternalServer(
	ctx context.Context,
	rr *rand.Rand,
	cfg proto.Config,
	treeFanOut int, // fanout of multicast trees
) *ExternalServer {

	discovery := discovery.NewDiscoveryService(
		ctx,
		discovery.DefaultDiscoveryConfig,
		libnet.TCPNet{},
		*cfg.Validator_config,
	)
	router := router.NewBasicRouter(ctx, discovery, treeFanOut)
	broadcast := broadcast.NewBroadcastService(
		ctx,
		*cfg.Broadcast_config,
		libnet.TCPNet{},
		runtime.NewRandFromRoot(),
		cfg.Validator_config.Private_info,
		router,
		broadcast.AcceptEverything{},
	)
	return &ExternalServer{
		discovery: discovery,
		broadcast: broadcast,
	}
}

func (s *ExternalServer) Start(ctx context.Context, address string) {
	s.l = rpcutil.Start(
		ctx,
		libnet.TCPNet{},
		address,
		proto.Discovery1_Server{Srv: s.discovery},
		proto.Broadcast1_Server{Srv: s.broadcast},
	)
}

func (s *ExternalServer) Close() {
	s.l.Close()
}
