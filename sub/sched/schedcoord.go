package sched

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	proto "github.com/groundhogchain/protocol-go"
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

func DialSchedulerCoordClient(ctx context.Context, dialer libnet.AddressDialer, address string) (SchedulerCoordConn, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewSchedulerCoordClient(ctx, sock), nil
}

func NewSchedulerCoordClient(ctx context.Context, sock net.Conn) *SchedulerCoordClient {
	driver := rpcutil.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &SchedulerCoordClient{
		SchedulerCoord1_Client: proto.SchedulerCoord1_Client{Send: driver, Ctx: ctx},
		driver:                 driver,
	}
}

func (cc *SchedulerCoordClient) Close() {
	cc.driver.Close()
}
