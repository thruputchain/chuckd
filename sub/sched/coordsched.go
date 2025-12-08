package sched

import (
	"context"
	"net"

	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type CoordSchedulerService struct {
	vetted chan proto.SignedTransactionBatch
	nr     *NominationReconstructor
}

func NewCoordSchedulerService() *CoordSchedulerService {
	return &CoordSchedulerService{
		vetted: make(chan proto.SignedTransactionBatch, 1),
		nr:     NewNominationReconstructor(),
	}
}

func (x *CoordSchedulerService) ReceiveVetted() proto.SignedTransactionBatch {
	return <-x.vetted
}

func (x *CoordSchedulerService) Coordscheduler_null(ctx context.Context) error {
	return nil
}

func (x *CoordSchedulerService) Coordscheduler_notify_vetted(ctx context.Context, txs proto.SignedTransactionBatch) error {
	x.vetted <- txs
	return nil
}

func (x *CoordSchedulerService) Coordscheduler_finish_nomination(ctx context.Context, chunk proto.SchedulerNominationChunk) error {
	x.nr.AddChunk(ctx, chunk)
	return nil
}

func (x *CoordSchedulerService) UpdateNoPartialNominationsBelow(ctx context.Context, height proto.BlockHeight) {
	x.nr.UpdateNoPartialNominationsBelow(ctx, height)
}

func (x *CoordSchedulerService) WaitForPartialNomination(
	ctx context.Context,
	id proto.BlockID,
	height proto.BlockHeight,
) (proto.Nomination, proto.BlockID, error) {
	return x.nr.WaitForPartialNomination(ctx, id, height)
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
	driver := librpc.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &CoordSchedulerClient{
		CoordScheduler1_Client: proto.CoordScheduler1_Client{Send: driver, Ctx: ctx},
		driver:                 driver,
	}
}

func (cc *CoordSchedulerClient) Close() {
	cc.driver.Close()
}
