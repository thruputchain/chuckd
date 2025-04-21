package coord

import (
	"context"

	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (svc *CoordinatorService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.Coordinator1_Server{Srv: svc})
}
