package debug

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type DebugClient struct {
	proto.Debug1_Client
	driver *rpc.Driver
}

func DialDebugClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*DebugClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewDebugClient(ctx, sock), nil
}

func NewDebugClient(ctx context.Context, sock net.Conn) *DebugClient {
	driver := rpcutil.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &DebugClient{
		Debug1_Client: proto.Debug1_Client{Send: driver, Ctx: ctx},
		driver:        driver,
	}
}

func (cc *DebugClient) Close() {
	cc.driver.Close()
}
