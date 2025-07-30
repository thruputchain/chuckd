package coord

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type DbCoordClient struct {
	proto.DbCoord1_Client
	driver *rpc.Driver
}

func DialDbCoordClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*DbCoordClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewDbCoordClient(ctx, sock), nil
}

func NewDbCoordClient(ctx context.Context, sock net.Conn) *DbCoordClient {
	driver := rpcutil.NewDriverSerial(ctx, rpcutil.NewRpcMsgPool(), sock)
	go driver.Go()

	return &DbCoordClient{
		DbCoord1_Client: proto.DbCoord1_Client{Send: driver, Ctx: ctx},
		driver:          driver,
	}
}

func (cc *DbCoordClient) Close() {
	cc.driver.Close()
}
