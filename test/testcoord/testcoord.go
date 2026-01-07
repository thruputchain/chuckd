package testcoord

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type TestCoord struct {
}

func NewTestCoord() *TestCoord {
	return &TestCoord{}
}

func (x *TestCoord) Coordscheduler_null(ctx context.Context) error {
	return nil
}

func (x *TestCoord) Coordscheduler_notify_vetted(ctx context.Context, txs proto.ParallelSignedTransactionBatch) error {
	return nil
}

func (x *TestCoord) Close() {}
