package coordinator

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
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
	liblog.Infof(ctx, "starting server on network %s at address %s", net, address)
	s.l = librpc.StartSerial(ctx, net, netutil.MaxMessageSize, rpcutil.BaseDriverRegime, address, s.srv...)
}

func (s *Server) Close() {
	s.Stop()
}

func (s *Server) Stop() {
	if s.l == nil {
		return
	}
	_ = s.l.Close()
	s.l = nil
}
