package repairkit

import (
	"context"
	"net"

	"github.com/thruputchain/chuckd/util/netutil"
	"github.com/thruputchain/chuckd/util/rpcutil"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/librpc"
	proto "github.com/thruputchain/thruput-protocol-go"
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

func DialRepairClient(ctx context.Context, dialer libnet.Dialer, addr string) (*RepairClient, error) {
	sock, err := dialer.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return NewRepairClient(context.WithoutCancel(ctx), sock), nil
}

func NewRepairClient(ctx context.Context, sock net.Conn) *RepairClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &RepairClient{
		Repair1_Client: proto.Repair1_Client{Send: rpcutil.LogReplyDecodeErrors(driver), Ctx: ctx},
		driver:         driver,
	}
}

func (cc *RepairClient) Close() {
	cc.driver.Close()
}
