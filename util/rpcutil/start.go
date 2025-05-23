package rpcutil

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/xdrpp/goxdr/rpc"
	"github.com/xdrpp/goxdr/xdr"
)

func NewDriver(ctx context.Context, t rpc.Transport) *rpc.Driver {
	driver := rpc.NewDriver(ctx, t)
	driver.Lock = nil // process all calls in parallel
	_ = log.Writer
	// driver.Log = log.Writer()
	driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
	return driver
}

func Start(ctx context.Context, n libnet.Network, addr string, srv ...xdr.XdrSrv) net.Listener {
	l, err := n.ListenAddress(addr)
	must.NoError(ctx, err)
	go func() {
		defer l.Close()
		for {
			sock, err := l.Accept()
			must.NoError(ctx, err)
			driver := NewDriver(ctx, rpc.NewStreamTransport(sock))
			for _, srv := range srv {
				driver.Register(srv)
			}
			go driver.Go()
		}
	}()
	return l
}
