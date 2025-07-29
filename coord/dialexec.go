package coord

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type ExecutorCoordClient struct {
	proto.ExecutorCoord1_Client
	driver *rpc.Driver
}

func DialExecutorCoordClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*ExecutorCoordClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewExecutorCoordClient(ctx, sock), nil
}

func NewExecutorCoordClient(ctx context.Context, sock net.Conn) *ExecutorCoordClient {
	driver := rpcutil.NewDriverSerial(ctx, rpc.NewMsgArena(), sock)
	go driver.Go()

	return &ExecutorCoordClient{
		ExecutorCoord1_Client: proto.ExecutorCoord1_Client{Send: driver, Ctx: ctx},
		driver:                driver,
	}
}

func (cc *ExecutorCoordClient) Close() {
	cc.driver.Close()
}
