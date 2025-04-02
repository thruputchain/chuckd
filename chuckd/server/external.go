// Package server implements a server that hosts all external APIs on the same address.
package server

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/discovery"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type ExternalServer struct {
	l         net.Listener
	discovery *discovery.DiscoveryService
}

func NewExternalServer(
	ctx context.Context,
	addr string,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
) *ExternalServer {
	log.Infof("starting external apis on %s", addr)
	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)
	return &ExternalServer{
		l:         l,
		discovery: discovery.NewDiscoveryService(ctx, privInfo, ourSlices),
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

		go driver.Go()
	}()
}

func (s *ExternalServer) Close() {
	s.l.Close()
}
