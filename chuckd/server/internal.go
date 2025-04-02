package server

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type InternalServer struct {
	l     net.Listener
	coord *coord.CoordinatorService
}

func NewInternalServer(
	ctx context.Context,
	addr string,
) *InternalServer {
	log.Infof("starting internal apis on %s", addr)
	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)
	return &InternalServer{
		l:     l,
		coord: coord.NewCoordinatorService(ctx),
	}
}

func (s *InternalServer) Start(ctx context.Context) {
	go func() {
		sock, err := s.l.Accept()
		must.NoError(ctx, err)
		driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
		driver.Lock = nil // process all calls in parallel
		driver.Log = log.Writer()
		driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)

		driver.Register(proto.Coordinator1_Server{Srv: s.coord})

		go driver.Go()
	}()
}

func (s *InternalServer) Close() {
	s.l.Close()
}
