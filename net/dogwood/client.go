package dogwood

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type DogwoodClientCloser interface {
	proto.Dogwood1
	Close()
}

type DogwoodClient struct {
	proto.Dogwood1_Client
	driver *rpc.Driver
}

func DialDogwoodClient(ctx context.Context, dialer libnet.AddressDialer, mp *rpc.MsgPool, addr string) (*DogwoodClient, error) {
	sock, err := dialer.DialAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewDogwoodClient(ctx, mp, sock), nil
}

func NewDogwoodClient(ctx context.Context, mp *rpc.MsgPool, sock net.Conn) *DogwoodClient {
	driver := rpcutil.NewDriverSerial(ctx, mp, sock)
	go driver.Go()

	return &DogwoodClient{
		Dogwood1_Client: proto.Dogwood1_Client{Send: driver, Ctx: ctx},
		driver:          driver,
	}
}

func (cc *DogwoodClient) Close() {
	cc.driver.Close()
}
