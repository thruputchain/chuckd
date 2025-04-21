package broadcast

import (
	"context"

	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (svc *BroadcastService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.Broadcast1_Server{Srv: svc})
}
