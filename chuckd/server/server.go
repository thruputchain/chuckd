package server

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type Server struct {
	internal *InternalServer
	external *ExternalServer
}

func NewServer(
	ctx context.Context,
	inaddr string,
	exaddr string,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
) *Server {
	return &Server{
		internal: NewInternalServer(ctx, inaddr),
		external: NewExternalServer(ctx, exaddr, privInfo, ourSlices),
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
