package advisory

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type AdvisoryClient struct {
	proto.Advisory1_Client
	driver *rpc.Driver
}

func DialAdvisoryClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*AdvisoryClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewAdvisoryClient(ctx, sock), nil
}

func NewAdvisoryClient(ctx context.Context, sock net.Conn) *AdvisoryClient {
	driver := rpcutil.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &AdvisoryClient{
		Advisory1_Client: proto.Advisory1_Client{Send: driver, Ctx: ctx},
		driver:           driver,
	}
}

func (cc *AdvisoryClient) Close() {
	cc.driver.Close()
}
