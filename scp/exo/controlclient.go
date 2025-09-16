package exo

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SCPControlClient struct {
	proto.SCP_Control1_Client
	driver *rpc.Driver
}

func DialSCPControlClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*SCPControlClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewSCPControlClient(ctx, sock), nil
}

func NewSCPControlClient(ctx context.Context, sock net.Conn) *SCPControlClient {
	driver := rpcutil.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &SCPControlClient{
		SCP_Control1_Client: proto.SCP_Control1_Client{Send: driver, Ctx: ctx},
		driver:              driver,
	}
}

func (cc *SCPControlClient) Close() {
	cc.driver.Close()
}
