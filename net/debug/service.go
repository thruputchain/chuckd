package debug

import (
	"context"

	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
)

type DebugService struct{}

func NewDebugService() *DebugService {
	return &DebugService{}
}

func (x *DebugService) Debug_ready(ctx context.Context) error {
	return nil
}

func (x *DebugService) Start(ctx context.Context, n libnet.Network, address string) {
	l := librpc.StartSerial(ctx, n, netutil.MaxMessageSize, rpcutil.BaseDriverRegime, address, proto.Debug1_Server{Srv: x})
	liblog.Infof(ctx, "debug service is listening on %v", l.Addr().String())
}
