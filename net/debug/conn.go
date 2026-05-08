package debug

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type DebugClient struct {
	proto.Debug1_Client
	driver *rpc.Driver
}

func DialDebugClient(ctx context.Context, dialer libnet.Dialer, address string) (*DebugClient, error) {
	sock, err := dialer.Dial(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewDebugClient(ctx, sock), nil
}

func NewDebugClient(ctx context.Context, sock net.Conn) *DebugClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &DebugClient{
		Debug1_Client: proto.Debug1_Client{Send: driver, Ctx: ctx},
		driver:        driver,
	}
}

func (cc *DebugClient) Close() {
	cc.driver.Close()
}
