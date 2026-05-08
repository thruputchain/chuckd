package echo_bench

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	"github.com/xdrpp/goxdr/rpc"
)

type EchoClientCloser interface {
	Echo1
	Close()
}

type EchoClient struct {
	Echo1_Client
	driver *rpc.Driver
}

func DialEchoClient(ctx context.Context, dialer libnet.Dialer, addr string) (*EchoClient, error) {
	sock, err := dialer.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewEchoClient(ctx, sock), nil
}

func NewEchoClient(ctx context.Context, sock net.Conn) *EchoClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &EchoClient{
		Echo1_Client: Echo1_Client{Send: driver, Ctx: ctx},
		driver:       driver,
	}
}

func (cc *EchoClient) Close() {
	cc.driver.Close()
}
