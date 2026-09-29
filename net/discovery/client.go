package discovery

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

type DiscoveryClient struct {
	proto.Discovery1_Client
	driver *rpc.Driver
}

func DialDiscoveryClient(ctx context.Context, dialer libnet.Dialer, address string) (*DiscoveryClient, error) {
	sock, err := dialer.Dial(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewDiscoveryClient(ctx, sock), nil
}

func NewDiscoveryClient(ctx context.Context, sock net.Conn) *DiscoveryClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &DiscoveryClient{
		Discovery1_Client: proto.Discovery1_Client{Send: rpcutil.LogReplyDecodeErrors(driver), Ctx: ctx},
		driver:            driver,
	}
}

func (cc *DiscoveryClient) Close() {
	cc.driver.Close()
}
