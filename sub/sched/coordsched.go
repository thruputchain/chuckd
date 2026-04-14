package sched

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type CoordSchedulerService struct {
	vetted chan proto.ParallelSignedTransactionBatch
	done   chan struct{}
	once   sync.Once
}

func NewCoordSchedulerService() *CoordSchedulerService {
	return &CoordSchedulerService{
		vetted: make(chan proto.ParallelSignedTransactionBatch, 1),
		done:   make(chan struct{}),
	}
}

func (x *CoordSchedulerService) ReceiveVetted(ctx context.Context) (proto.ParallelSignedTransactionBatch, bool) {
	select {
	case <-ctx.Done():
		return proto.ParallelSignedTransactionBatch{}, false
	case <-x.done:
		return proto.ParallelSignedTransactionBatch{}, false
	case batch := <-x.vetted:
		return batch, true
	}
}

func (x *CoordSchedulerService) Stop() {
	x.once.Do(func() {
		close(x.done)
	})
}

func (x *CoordSchedulerService) Coordscheduler_null(ctx context.Context) error {
	return nil
}

func (x *CoordSchedulerService) Coordscheduler_notify_vetted(ctx context.Context, txs proto.ParallelSignedTransactionBatch) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-x.done:
		return errors.New("coord scheduler service is stopped")
	case x.vetted <- txs:
		return nil
	}
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
