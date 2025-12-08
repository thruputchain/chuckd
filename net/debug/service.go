package debug

import (
	"context"

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
	l := librpc.StartSerial(ctx, n, address, proto.Debug1_Server{Srv: x})
	liblog.Infof("debug service is listening on %v", l.Addr().String())
}
