package rpcutil

import (
	"bytes"
	"context"
	"fmt"
	"net"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/xdrpp/goxdr/rpc"
	"github.com/xdrpp/goxdr/xdr"
)

func NewDriver(ctx context.Context, t rpc.Transport) *rpc.Driver {
	driver := rpc.NewDriver(ctx, t)
	// don't process in parallel, as the per-call goroutine is expensive
	// driver.Lock = nil // process all calls in parallel
	_ = liblog.Writer
	// driver.Log = log.Writer()
	driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
	return driver
}

func Start(ctx context.Context, n libnet.Network, addr string, srv ...xdr.XdrSrv) net.Listener {
	l, err := n.ListenAddress(ctx, addr)
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

func srvList(srv []xdr.XdrSrv) string {
	var w bytes.Buffer
	for _, s := range srv {
		fmt.Fprintf(&w, "%v, ", s.ProgName())
	}
	return w.String()
}
