package system

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/runtime"
	proto "github.com/groundhogchain/protocol-go"
)

type Server struct {
	internal *InternalServer
	external *ExternalServer
}

func NewServer(
	ctx context.Context,
	rr *rand.Rand,
	cfg proto.Config,
	treeFanOut int, // fanout of multicast trees
) *Server {
	return &Server{
		internal: NewInternalServer(ctx, runtime.NewRandFromRand(rr)),
		external: NewExternalServer(ctx, runtime.NewRandFromRand(rr), cfg, treeFanOut),
	}
}

func (s *Server) Start(ctx context.Context, inaddr string, exaddr string) {
	s.internal.Start(ctx, inaddr)
	s.external.Start(ctx, exaddr)
}

func (s *Server) Close() {
	s.internal.Close()
	s.external.Close()
}
