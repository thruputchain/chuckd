package sched

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

type CoordSchedulerService struct {
	vetted chan proto.ParallelSignedTransactionBatch
}

func NewCoordSchedulerService() *CoordSchedulerService {
	return &CoordSchedulerService{
		vetted: make(chan proto.ParallelSignedTransactionBatch, 1),
	}
}

func (x *CoordSchedulerService) ReceiveVetted() proto.ParallelSignedTransactionBatch {
	return <-x.vetted
}

func (x *CoordSchedulerService) Coordscheduler_null(ctx context.Context) error {
	return nil
}

func (x *CoordSchedulerService) Coordscheduler_notify_vetted(ctx context.Context, txs proto.ParallelSignedTransactionBatch) error {
	x.vetted <- txs
	return nil
}

// client

type CoordSchedulerConn interface {
	proto.CoordScheduler1
	Close()
}

type CoordSchedulerClient struct {
	proto.CoordScheduler1_Client
	driver *rpc.Driver
}

func DialCoordSchedulerClient(ctx context.Context, dialer libnet.AddressDialer, address string) (CoordSchedulerConn, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewCoordSchedulerClient(ctx, sock), nil
}

func NewCoordSchedulerClient(ctx context.Context, sock net.Conn) *CoordSchedulerClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &CoordSchedulerClient{
		CoordScheduler1_Client: proto.CoordScheduler1_Client{Send: driver, Ctx: ctx},
		driver:                 driver,
	}
}

func (cc *CoordSchedulerClient) Close() {
	cc.driver.Close()
}
