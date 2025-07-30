package rpcutil

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/xdrpp/goxdr/rpc"
	"github.com/xdrpp/goxdr/xdr"
)

func NewDriverSerial(ctx context.Context, sock net.Conn) *rpc.Driver {
	mp := NewRpcMsgPool()
	driver := NewDriver(ctx, mp, NewRpcCallPool(), rpc.NewStreamTransportWithPool(sock, mp))
	_ = liblog.Writer
	// driver.Log = log.Writer()
	driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
	return driver
}

func NewDriverParallel(ctx context.Context, sock net.Conn) *rpc.Driver {
	mp := NewRpcMsgPool()
	driver := NewDriver(ctx, mp, NewRpcCallPool(), rpc.NewStreamTransportWithPool(sock, mp))
	driver.Lock = nil // process all calls in parallel
	_ = liblog.Writer
	// driver.Log = log.Writer()
	driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
	return driver
}

func StartSerial(ctx context.Context, n libnet.Network, addr string, srv ...xdr.XdrSrv) net.Listener {
	l, err := n.ListenAddress(ctx, addr)
	must.NoError(ctx, err)
	go func() {
		defer l.Close()
		for {
			sock, err := l.Accept()
			must.NoError(ctx, err)
			driver := NewDriverSerial(ctx, sock)
			for _, srv := range srv {
				driver.Register(srv)
			}
			go driver.Go()
		}
	}()
	return l
}

func StartParallel(ctx context.Context, n libnet.Network, addr string, srv ...xdr.XdrSrv) net.Listener {
	l, err := n.ListenAddress(ctx, addr)
	must.NoError(ctx, err)
	go func() {
		defer l.Close()
		for {
			sock, err := l.Accept()
			must.NoError(ctx, err)
			driver := NewDriverParallel(ctx, sock)
			for _, srv := range srv {
				driver.Register(srv)
			}
			go driver.Go()
		}
	}()
	return l
}
