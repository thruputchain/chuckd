package hoglock

import (
	"context"
	"net"

	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type HogLockConn interface {
	proto.HogLock1
	Close()
}

type HogLockClient struct {
	proto.HogLock1_Client
	driver *rpc.Driver
}

func DialHogLockClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*HogLockClient, error) {
	sock, err := dialer.DialAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewHogLockClient(ctx, sock), nil
}

func NewHogLockClient(ctx context.Context, sock net.Conn) *HogLockClient {
	driver := librpc.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &HogLockClient{
		HogLock1_Client: proto.HogLock1_Client{Send: driver, Ctx: ctx},
		driver:          driver,
	}
}

func (cc *HogLockClient) Close() {
	cc.driver.Close()
}
