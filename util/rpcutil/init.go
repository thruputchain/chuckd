package rpcutil

import "github.com/xdrpp/goxdr/rpc"

func NewRpcMsgPool() rpc.MessagePool {
	return rpc.NewMsgArenaCap(5000)
}
