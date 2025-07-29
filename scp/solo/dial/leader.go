package dial

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SoloLeaderClient struct {
	proto.SoloLeader1_Client
	driver *rpc.Driver
}

func DialSoloLeaderClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*SoloLeaderClient, error) {
	sock, err := dialer.DialAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewSoloLeaderClient(ctx, sock), nil
}

func NewSoloLeaderClient(ctx context.Context, sock net.Conn) *SoloLeaderClient {
	driver := rpcutil.NewDriverSerial(ctx, rpc.NewMessagePool(), sock)
	go driver.Go()

	return &SoloLeaderClient{
		SoloLeader1_Client: proto.SoloLeader1_Client{Send: driver, Ctx: ctx},
		driver:             driver,
	}
}

func (cc *SoloLeaderClient) Close() {
	cc.driver.Close()
}
