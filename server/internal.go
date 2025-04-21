package server

import (
	"context"
	"math/rand"
	"net"

	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type InternalServer struct {
	l     net.Listener
	coord *coord.CoordinatorService
}

func NewInternalServer(
	ctx context.Context,
	rr *rand.Rand,
) *InternalServer {
	return &InternalServer{
		coord: coord.NewCoordinatorService(ctx, libnet.TCPNet{}, runtime.NewRandFromRand(rr)),
	}
}

func (s *InternalServer) Start(ctx context.Context, address string) {
	s.l = rpcutil.Start(
		ctx,
		libnet.TCPNet{},
		address,
		proto.Coordinator1_Server{Srv: s.coord},
	)
}

func (s *InternalServer) Close() {
	s.l.Close()
}
