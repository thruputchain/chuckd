package shard

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type ShardClient struct {
	proto.Shard1_Client
	driver *rpc.Driver
}

func NewShardClient(ctx context.Context, addr string) *ShardClient {

	sock, err := net.Dial("tcp", addr)
	must.NoError(ctx, err)

	driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
	go driver.Go()

	return &ShardClient{
		Shard1_Client: proto.Shard1_Client{Send: driver, Ctx: ctx},
		driver:        driver,
	}
}

func (cc *ShardClient) Close() {
	cc.driver.Close()
}
