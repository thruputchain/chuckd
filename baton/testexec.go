package baton

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type TestExec struct {
	db                 *TestDb
	lk                 sync.Mutex
	openedID           *proto.BlockID
	nextValidateResult proto.ExecResult
}

func NewTestExec(db *TestDb) *TestExec {
	return &TestExec{
		db:                 db,
		nextValidateResult: proto.ExecResultOK,
	}
}

func (x *TestExec) Executorcoord_null(context.Context) error {
	return nil
}

func (x *TestExec) Executorcoord_propose_transaction(ctx context.Context, blockID proto.BlockID, stx proto.SignedTransaction) (proto.ExecProposeResult, error) {
	x.db.assertOpen(ctx, blockID)
	x.lk.Lock()
	must.Assertf(ctx, x.openedID != nil && *x.openedID == blockID, "exec block not open")
	x.lk.Unlock()
	x.db.addSignedTx(ctx, blockID, &stx)
	result := proto.ExecProposeResult{Result: proto.ExecResultOK}
	*result.Proposal_data() = testNondeterministicData
	return result, nil
}

func (x *TestExec) Control_SetNextValidateResult(r proto.ExecResult) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.nextValidateResult = r
}

func (x *TestExec) Executorcoord_validate_transaction(ctx context.Context, blockID proto.BlockID, ptx proto.ProposedTransaction) (proto.ExecResult, error) {
	x.db.assertOpen(ctx, blockID)
	x.lk.Lock()
	must.Assertf(ctx, x.openedID != nil && *x.openedID == blockID, "exec block not open")
	x.lk.Unlock()
	x.db.addProposedTx(ctx, blockID, &ptx)
	return proto.ExecResultOK, nil
}

func (x *TestExec) Executorcoord_open_proposal_block(ctx context.Context, blockID proto.BlockID) (proto.ExecResult, error) {
	x.db.assertOpen(ctx, blockID)
	x.lk.Lock()
	defer x.lk.Unlock()
	must.Assertf(ctx, x.openedID == nil, "executor block already open")
	x.openedID = &blockID
	return proto.ExecResultOK, nil
}

func (x *TestExec) Executorcoord_open_validation_block(ctx context.Context, blockID proto.BlockID) (proto.ExecResult, error) {
	x.db.assertOpen(ctx, blockID)
	x.lk.Lock()
	defer x.lk.Unlock()
	must.Assertf(ctx, x.openedID == nil, "executor block already open")
	x.openedID = &blockID
	return proto.ExecResultOK, nil
}

func (x *TestExec) Executorcoord_finish_block(ctx context.Context, blockID proto.BlockID) error {
	x.db.assertOpen(ctx, blockID)
	x.lk.Lock()
	defer x.lk.Unlock()
	x.openedID = nil
	return nil
}

func (x *TestExec) Executorcoord_shutdown(ctx context.Context) error {
	return nil
}

func (x *TestExec) Close() error {
	return nil
}
