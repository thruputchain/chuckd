package rpcutil

import "github.com/xdrpp/goxdr/rpc"

var BaseDriverRegime = rpc.DriverRegime{
	RecvQueueLen: 10,
	SendQueueLen: 10,
	NumProc:      10,
}
