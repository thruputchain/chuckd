package coord

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type CoordinatorClient struct {
	proto.Coordinator1_Client
	driver *rpc.Driver
}

func NewCoordinatorClient(ctx context.Context, addr string) *CoordinatorClient {

	sock, err := net.Dial("tcp", addr)
	must.NoError(ctx, err)

	driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
	go driver.Go()

	return &CoordinatorClient{
		Coordinator1_Client: proto.Coordinator1_Client{Send: driver, Ctx: ctx},
		driver:              driver,
	}
}

func (cc *CoordinatorClient) Close() {
	cc.driver.Close()
}
