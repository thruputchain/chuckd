package broadcast

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type BroadcastClient struct {
	proto.Broadcast1_Client
	driver *rpc.Driver
}

func DialBroadcastClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*BroadcastClient, error) {
	sock, err := dialer.DialAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewBroadcastClient(ctx, sock), nil
}

func DialBroadcastClientTCP(ctx context.Context, addr string) *BroadcastClient {
	sock, err := net.Dial("tcp", addr)
	must.NoError(ctx, err)
	return NewBroadcastClient(ctx, sock)
}

func NewBroadcastClient(ctx context.Context, sock net.Conn) *BroadcastClient {
	driver := rpcutil.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &BroadcastClient{
		Broadcast1_Client: proto.Broadcast1_Client{Send: driver, Ctx: ctx},
		driver:            driver,
	}
}

func (cc *BroadcastClient) Close() {
	cc.driver.Close()
}
