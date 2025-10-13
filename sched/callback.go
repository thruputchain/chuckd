package sched

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type SchedulerCallbackService struct {
	vetted chan proto.SignedTransactionBatch
}

func NewSchedulerCallbackService() *SchedulerCallbackService {
	return &SchedulerCallbackService{
		vetted: make(chan proto.SignedTransactionBatch, 1),
	}
}

func (x *SchedulerCallbackService) ReceiveVetted() proto.SignedTransactionBatch {
	// XXX: buffer?
	return <-x.vetted
}

func (x *SchedulerCallbackService) Coordscheduler_null(ctx context.Context) error {
	return nil
}

func (x *SchedulerCallbackService) Coordscheduler_notify_vetted(ctx context.Context, txs proto.SignedTransactionBatch) error {
	x.vetted <- txs
	return nil
}
