// Package server implements a server that hosts all external APIs on the same address.
package server

import (
	"context"
	"math/rand"
	"net"

	"github.com/groundhogchain/chuckd/broadcast"
	"github.com/groundhogchain/chuckd/discovery"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/runtime"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type ExternalServer struct {
	l         net.Listener
	discovery *discovery.DiscoveryService
	broadcast *broadcast.BroadcastService
}

func NewExternalServer(
	ctx context.Context,
	rr *rand.Rand,
	addr string,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	maxFanOut int, // maximum fanout of multicast trees
) *ExternalServer {

	log.Infof("starting external apis on %s", addr)
	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)

	discovery := discovery.NewDiscoveryService(ctx, privInfo, ourSlices)
	router := router.NewBasicRouter(ctx, discovery, maxFanOut)
	broadcast := broadcast.NewBroadcastService(ctx, runtime.NewRandFromRand(rr), privInfo, ourSlices, router)
	return &ExternalServer{
		l:         l,
		discovery: discovery,
		broadcast: broadcast,
	}
}

func (s *ExternalServer) Start(ctx context.Context) {
	go func() {
		sock, err := s.l.Accept()
		must.NoError(ctx, err)
		driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
		driver.Lock = nil // process all calls in parallel
		driver.Log = log.Writer()
		driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)

		driver.Register(proto.Discovery1_Server{Srv: s.discovery})
		driver.Register(proto.Broadcast1_Server{Srv: s.broadcast})

		go driver.Go()
	}()
}

func (s *ExternalServer) Close() {
	s.l.Close()
}
