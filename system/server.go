package system

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/xdrpp/goxdr/xdr"
)

type Server struct {
	l   net.Listener
	srv []xdr.XdrSrv
}

func NewServer(ctx context.Context, srv ...xdr.XdrSrv) *Server {
	return &Server{srv: srv}
}

func (s *Server) Start(ctx context.Context, net libnet.Network, address string) {
	s.l = rpcutil.Start(ctx, net, address, s.srv...)
}

func (s *Server) Close() {
	s.l.Close()
}
