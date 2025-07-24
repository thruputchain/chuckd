package discovery

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type DiscoveryClient struct {
	proto.Discovery1_Client
	driver *rpc.Driver
}

func DialDiscoveryClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*DiscoveryClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewDiscoveryClient(ctx, sock), nil
}

func NewDiscoveryClient(ctx context.Context, sock net.Conn) *DiscoveryClient {
	driver := rpcutil.NewDriverSerial(ctx, rpc.NewMsgPool(), sock)
	go driver.Go()

	return &DiscoveryClient{
		Discovery1_Client: proto.Discovery1_Client{Send: driver, Ctx: ctx},
		driver:            driver,
	}
}

func (cc *DiscoveryClient) Close() {
	cc.driver.Close()
}
