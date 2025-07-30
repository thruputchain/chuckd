package rpcutil

import (
	"context"

	"github.com/xdrpp/goxdr/rpc"
)

func NewRpcMsgPool() rpc.MessagePool {
	return rpc.NewMsgArena(5000)
}

func NewRpcCallPool() rpc.CallPool {
	return rpc.NewCallArena(5000)
}

func NewDriver(ctx context.Context, mp rpc.MessagePool, cp rpc.CallPool, t rpc.Transport) *rpc.Driver {
	return rpc.NewDriverTuned(ctx, mp, cp, t, 10, 10, 3)
}
