package discovery

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type DiscoveryClient struct {
	proto.Discovery1_Client
	driver *rpc.Driver
}

func NewDiscoveryClient(ctx context.Context, addr string) *DiscoveryClient {

	sock, err := net.Dial("tcp", addr)
	must.NoError(ctx, err)

	driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
	go driver.Go()

	return &DiscoveryClient{
		Discovery1_Client: proto.Discovery1_Client{Send: driver, Ctx: ctx},
		driver:            driver,
	}
}

func (cc *DiscoveryClient) Close() {
	cc.driver.Close()
}
