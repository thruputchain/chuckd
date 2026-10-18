package debug

import (
	"context"

	"github.com/thruputchain/chuckd/util/netutil"
	"github.com/thruputchain/chuckd/util/rpcutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/librpc"
	proto "github.com/thruputchain/thruput-protocol-go"
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
