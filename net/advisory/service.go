package advisory

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type AdvisoryService struct{}

func NewAdvisoryService() *AdvisoryService {
	return &AdvisoryService{}
}

func (x *AdvisoryService) Advisory_ready(ctx context.Context) error {
	return nil
}

func (x *AdvisoryService) Start(ctx context.Context, n libnet.Network, address string) {
	l := rpcutil.StartSerial(ctx, n, address, proto.Advisory1_Server{Srv: x})
	liblog.Infof("advisory service is listening on %v", l.Addr().String())
}
