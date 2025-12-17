package repairkit

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

type RepairClientCloser interface {
	proto.Repair1
	Close()
}

type RepairClient struct {
	proto.Repair1_Client
	driver *rpc.Driver
}

func DialRepairClient(ctx context.Context, dialer libnet.AddressDialer, addr string) (*RepairClient, error) {
	sock, err := dialer.DialAddress(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewRepairClient(context.WithoutCancel(ctx), sock), nil
}

func NewRepairClient(ctx context.Context, sock net.Conn) *RepairClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &RepairClient{
		Repair1_Client: proto.Repair1_Client{Send: driver, Ctx: ctx},
		driver:         driver,
	}
}

func (cc *RepairClient) Close() {
	cc.driver.Close()
}
