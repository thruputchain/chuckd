package catchup

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type CatchupClient struct {
	proto.Catchup1_Client
	driver *rpc.Driver
}

func DialCatchupClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*CatchupClient, error) {
	sock, err := dialer.DialAddress(addr)
	if err != nil {
		return nil, err
	}
	return NewCatchupClient(ctx, sock), nil
}

func NewCatchupClient(ctx context.Context, sock net.Conn) *CatchupClient {
	driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
	go driver.Go()

	return &CatchupClient{
		Catchup1_Client: proto.Catchup1_Client{Send: driver, Ctx: ctx},
		driver:          driver,
	}
}

func (cc *CatchupClient) Close() {
	cc.driver.Close()
}
