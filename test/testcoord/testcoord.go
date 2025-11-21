package testcoord

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/sched"
	proto "github.com/groundhogchain/protocol-go"
)

type TestCoord struct {
	nr *sched.NominationReconstructor
}

func NewTestCoord() *TestCoord {
	return &TestCoord{
		nr: sched.NewNominationReconstructor(),
	}
}

func (x *TestCoord) Coordscheduler_null(ctx context.Context) error {
	return nil
}

func (x *TestCoord) Coordscheduler_notify_vetted(ctx context.Context, txs proto.SignedTransactionBatch) error {
	return nil
}

func (x *TestCoord) Coordscheduler_finish_nomination(ctx context.Context, chunk proto.SchedulerNominationChunk) error {
	x.nr.AddChunk(ctx, chunk)
	return nil
}

func (x *TestCoord) UpdateNoNominationsBelow(ctx context.Context, height proto.BlockHeight) {
	x.nr.UpdateNoNominationsBelow(ctx, height)
}

func (x *TestCoord) WaitForNomination(
	ctx context.Context,
	height proto.BlockHeight,
) (proto.Nomination, proto.BlockID, error) {
	return x.nr.WaitForNomination(ctx, height)
}

func (x *TestCoord) Close() {}
