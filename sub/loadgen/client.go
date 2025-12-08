package loadgen

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type LoadgenClient struct {
	proto.LoadgenInterfaceV1_Client
	driver *rpc.Driver
}

func DialLoadgenClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*LoadgenClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewLoadgenClient(ctx, sock), nil
}

func NewLoadgenClient(ctx context.Context, sock net.Conn) *LoadgenClient {
	driver := rpcutil.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &LoadgenClient{
		LoadgenInterfaceV1_Client: proto.LoadgenInterfaceV1_Client{Send: driver, Ctx: ctx},
		driver:                    driver,
	}
}

func (cc *LoadgenClient) Close() {
	cc.driver.Close()
}
