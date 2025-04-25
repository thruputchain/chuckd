package dial

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SoloServerClient struct {
	proto.SoloServer1_Client
	driver *rpc.Driver
}

func DialSoloServerClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*SoloServerClient, error) {
	sock, err := dialer.DialAddress(addr)
	if err != nil {
		return nil, err
	}
	return NewSoloServerClient(ctx, sock), nil
}

func NewSoloServerClient(ctx context.Context, sock net.Conn) *SoloServerClient {
	driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
	go driver.Go()

	return &SoloServerClient{
		SoloServer1_Client: proto.SoloServer1_Client{Send: driver, Ctx: ctx},
		driver:             driver,
	}
}

func (cc *SoloServerClient) Close() {
	cc.driver.Close()
}
