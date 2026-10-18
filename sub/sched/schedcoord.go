package sched

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

type SchedulerCoordConn interface {
	proto.SchedulerCoord1
	Close()
}

type SchedulerCoordClient struct {
	proto.SchedulerCoord1_Client
	driver *rpc.Driver
}

func DialSchedulerCoordClient(ctx context.Context, dialer libnet.Dialer, address string) (SchedulerCoordConn, error) {
	sock, err := dialer.Dial(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewSchedulerCoordClient(ctx, sock), nil
}

func NewSchedulerCoordClient(ctx context.Context, sock net.Conn) *SchedulerCoordClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &SchedulerCoordClient{
		SchedulerCoord1_Client: proto.SchedulerCoord1_Client{Send: driver, Ctx: ctx},
		driver:                 driver,
	}
}

func (cc *SchedulerCoordClient) Close() {
	cc.driver.Close()
}
