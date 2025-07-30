package rpcutil

import (
	"context"

	"github.com/xdrpp/goxdr/rpc"
)

func NewRpcMsgPool() rpc.MessagePool {
	return rpc.NewMsgArenaCap(5000)
}

func NewDriver(ctx context.Context, mp rpc.MessagePool, t rpc.Transport) *rpc.Driver {
	return rpc.NewDriverTuned(ctx, mp, t, 10, 10, 3)
}
