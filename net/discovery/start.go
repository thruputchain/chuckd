package discovery

import (
	"context"

	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (svc *DiscoveryService) Start(ctx context.Context, address string) {
	rpcutil.StartSerial(ctx, svc.net, address, proto.Discovery1_Server{Srv: svc})
}
