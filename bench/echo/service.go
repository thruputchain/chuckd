package echo

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
)

type EchoService struct {
	net libnet.Network
}

func NewEchoService(
	ctx context.Context,
	net libnet.Network,
) *EchoService {
	return &EchoService{
		net: net,
	}
}

func (x *EchoService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, x.net, address, Echo1_Server{Srv: x})
}

func (x *EchoService) Null(context.Context) error {
	return nil
}

func (x *EchoService) Echo(_ context.Context, req Request) (Reply, error) {
	return Reply{Payload: req.Payload}, nil
}
