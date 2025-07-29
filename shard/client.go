package shard

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type ShardClient struct {
	proto.Shard1_Client
	driver *rpc.Driver
}

func DialShardClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*ShardClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewShardClient(ctx, sock), nil
}

func NewShardClient(ctx context.Context, sock net.Conn) *ShardClient {
	driver := rpcutil.NewDriverSerial(ctx, rpc.NewMsgArena(), sock)
	go driver.Go()

	return &ShardClient{
		Shard1_Client: proto.Shard1_Client{Send: driver, Ctx: ctx},
		driver:        driver,
	}
}

func (cc *ShardClient) Close() {
	cc.driver.Close()
}
