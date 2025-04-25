package dial

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SoloClientClient struct {
	proto.SoloClient1_Client
	driver *rpc.Driver
}

func DialSoloClientClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*SoloClientClient, error) {
	sock, err := dialer.DialAddress(addr)
	if err != nil {
		return nil, err
	}
	return NewSoloClientClient(ctx, sock), nil
}

func NewSoloClientClient(ctx context.Context, sock net.Conn) *SoloClientClient {
	driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
	go driver.Go()

	return &SoloClientClient{
		SoloClient1_Client: proto.SoloClient1_Client{Send: driver, Ctx: ctx},
		driver:             driver,
	}
}

func (cc *SoloClientClient) Close() {
	cc.driver.Close()
}
