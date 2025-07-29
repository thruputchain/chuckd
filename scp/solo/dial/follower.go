package dial

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SoloFollowerClient struct {
	proto.SoloFollower1_Client
	driver *rpc.Driver
}

func DialSoloFollowerClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*SoloFollowerClient, error) {
	sock, err := dialer.DialAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewSoloFollowerClient(ctx, sock), nil
}

func NewSoloFollowerClient(ctx context.Context, sock net.Conn) *SoloFollowerClient {
	driver := rpcutil.NewDriverSerial(ctx, rpc.NewMsgArena(), sock)
	go driver.Go()

	return &SoloFollowerClient{
		SoloFollower1_Client: proto.SoloFollower1_Client{Send: driver, Ctx: ctx},
		driver:               driver,
	}
}

func (cc *SoloFollowerClient) Close() {
	cc.driver.Close()
}
