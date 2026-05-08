package catchup

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type CatchupConn interface {
	proto.Catchup1
	Close()
}

type CatchupClient struct {
	proto.Catchup1_Client
	driver *rpc.Driver
}

func DialCatchupClient(ctx context.Context, dialer libnet.Dialer, addr string) (*CatchupClient, error) {
	sock, err := dialer.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewCatchupClient(ctx, sock), nil
}

func NewCatchupClient(ctx context.Context, sock net.Conn) *CatchupClient {
	driver := librpc.NewDriverSerial(ctx, sock, systemfixed.MaxBytesCatchupReply, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &CatchupClient{
		Catchup1_Client: proto.Catchup1_Client{Send: driver, Ctx: ctx},
		driver:          driver,
	}
}

func (cc *CatchupClient) Close() {
	cc.driver.Close()
}
