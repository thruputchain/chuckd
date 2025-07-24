package coord

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type CoordinatorClient struct {
	proto.Coordinator1_Client
	driver *rpc.Driver
}

func DialCoordinatorClientTCP(ctx context.Context, address string) (*CoordinatorClient, error) {
	sock, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return NewCoordinatorClient(ctx, sock), nil
}

func DialCoordinatorClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*CoordinatorClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewCoordinatorClient(ctx, sock), nil
}

func NewCoordinatorClient(ctx context.Context, sock net.Conn) *CoordinatorClient {
	driver := rpcutil.NewDriverSerial(ctx, rpc.NewMsgPool(), sock)
	go driver.Go()

	return &CoordinatorClient{
		Coordinator1_Client: proto.Coordinator1_Client{Send: driver, Ctx: ctx},
		driver:              driver,
	}
}

func (cc *CoordinatorClient) Close() {
	cc.driver.Close()
}
