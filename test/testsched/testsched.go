package testsched

import (
	"context"
	"sync"

	"github.com/groundhogchain/jay/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type TestScheduler struct {
	lk     sync.Mutex
	introd bool
}

func NewTestScheduler(ctx context.Context, tip proto.DbTip) *TestScheduler {
	return &TestScheduler{}
}

func (x *TestScheduler) Schedulercoord_null(context.Context) error {
	return nil
}

func (x *TestScheduler) Schedulercoord_intro_coord(ctx context.Context, card proto.CoordCard) (proto.CoordinatorRegistrationOutcome, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	must.Assert(ctx, !x.introd)
	x.introd = true
	return proto.CoordinatorRegistrationOK, nil
}

func (x *TestScheduler) Schedulercoord_submit_unvetted(ctx context.Context, txs proto.SignedTransactionBatch) error {
	panic("XXX")
}

func (x *TestScheduler) Schedulercoord_nominate(ctx context.Context, onto proto.DbTip) (proto.SchedulerNominationResult, error) {
	panic("XXX")
}

func (x *TestScheduler) Schedulercoord_validate(ctx context.Context, onto proto.DbTip, nomination proto.Nomination) (proto.SchedulerValidationResult, error) {
	panic("XXX")
}

func (x *TestScheduler) Schedulercoord_commit_block(ctx context.Context, blockID proto.BlockID) error {
	panic("XXX")
}

func (x *TestScheduler) Schedcoord_get_tip(ctx context.Context) (proto.DbTipResult, error) {
	panic("XXX")
}
