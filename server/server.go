package server

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
	inaddr string,
	exaddr string,
	vcfg proto.ValidatorConfig,
	maxFanOut int, // maximum fanout of multicast trees
) *Server {
	return &Server{
		internal: NewInternalServer(ctx, runtime.NewRandFromRand(rr), inaddr),
		external: NewExternalServer(ctx, runtime.NewRandFromRand(rr), exaddr, vcfg, maxFanOut),
	}
}

func (s *Server) Start(ctx context.Context) {
	s.internal.Start(ctx)
	s.external.Start(ctx)
}

func (s *Server) Close() {
	s.internal.Close()
	s.external.Close()
}
