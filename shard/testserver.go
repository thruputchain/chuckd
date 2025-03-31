package shard

import (
	"context"
	"log"
	"net"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type TestShardServer struct {
	l net.Listener
	s *TestShardService
}

func NewTestShardServer(ctx context.Context) *TestShardServer {
	s := NewTestShardService(ctx)
	l, err := net.Listen("tcp", "localhost:0")
	must.NoError(ctx, err)
	return &TestShardServer{l: l, s: s}
}

func (s *TestShardServer) Addr() string {
	return s.l.Addr().String()
}

func (s *TestShardServer) Start(ctx context.Context) {
	go func() {
		for {
			sock, err := s.l.Accept()
			must.NoError(ctx, err)
			driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
			driver.Lock = nil // process all calls in parallel
			driver.Log = log.Writer()
			driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
			driver.Register(proto.Shard1_Server{Srv: s.s})
			go driver.Go()
		}
	}()
}
