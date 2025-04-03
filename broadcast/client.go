package broadcast

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type BroadcastClient struct {
	proto.Broadcast1_Client
	driver *rpc.Driver
}

func NewBroadcastClient(ctx context.Context, addr string) *BroadcastClient {

	sock, err := net.Dial("tcp", addr)
	must.NoError(ctx, err)

	driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
	go driver.Go()

	return &BroadcastClient{
		Broadcast1_Client: proto.Broadcast1_Client{Send: driver, Ctx: ctx},
		driver:            driver,
	}
}

func (cc *BroadcastClient) Close() {
	cc.driver.Close()
}
