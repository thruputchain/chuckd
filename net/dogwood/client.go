package dogwood

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
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

func DialDogwoodClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*DogwoodClient, error) {
	sock, err := dialer.DialAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewDogwoodClient(context.WithoutCancel(ctx), sock), nil
}

func NewDogwoodClient(ctx context.Context, sock net.Conn) *DogwoodClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &DogwoodClient{
		Dogwood1_Client: proto.Dogwood1_Client{Send: driver, Ctx: ctx},
		driver:          driver,
	}
}

func (cc *DogwoodClient) Close() {
	cc.driver.Close()
}
