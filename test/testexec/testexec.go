package testexec

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/test/testdb"
	"github.com/groundhogchain/chuckd/test/testutil"
	proto "github.com/groundhogchain/protocol-go"
)

// TODO: add feature to fail validating certain txs, based on their contents

type TestExec struct {
	db   testdb.TestDb
	lk   sync.Mutex
	open map[proto.BlockID]int
}

func NewTestExec(db testdb.TestDb) *TestExec {
	return &TestExec{
		db:   db,
		open: map[proto.BlockID]int{},
	}
}

func (x *TestExec) Executorcoord_null(context.Context) error {
	return nil
}

func (x *TestExec) Executorcoord_propose_transaction(ctx context.Context, blockID proto.BlockID, stx proto.SignedTransaction) (proto.ExecProposeResult, error) {
	if db, ok := x.db.(interface{ IsOpenBlock(proto.BlockID) bool }); ok {
		must.Assertf(ctx, db.IsOpenBlock(blockID), "propose: db block not open")
	}
	x.lk.Lock()
	must.Assertf(ctx, x.open[blockID] > 0, "exec block not open")
	x.lk.Unlock()
	x.db.AddSignedTx(ctx, blockID, &stx)
	result := proto.ExecProposeResult{Result: proto.ExecResultOK}
	*result.Proposal_data() = testutil.TestNondeterministicData
	return result, nil
}

func (x *TestExec) Executorcoord_validate_transaction(ctx context.Context, blockID proto.BlockID, ptx proto.ProposedTransaction) (proto.ExecResult, error) {
	if db, ok := x.db.(interface{ IsOpenBlock(proto.BlockID) bool }); ok {
		must.Assertf(ctx, db.IsOpenBlock(blockID), "validate: db block not open")
	}
	x.lk.Lock()
	must.Assertf(ctx, x.open[blockID] > 0, "exec block not open")
	x.lk.Unlock()
	x.db.AddProposedTx(ctx, blockID, &ptx)
	return proto.ExecResultOK, nil
}

func (x *TestExec) Executorcoord_open_proposal_block(ctx context.Context, blockID proto.BlockID) (proto.ExecResult, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	// must.Assertf(ctx, x.open[blockID] == 0, "executor block already open")
	x.open[blockID] += 1
	return proto.ExecResultOK, nil
}

func (x *TestExec) Executorcoord_open_validation_block(ctx context.Context, blockID proto.BlockID) (proto.ExecResult, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	// must.Assertf(ctx, x.open[blockID] == 0, "executor block already open")
	x.open[blockID] += 1
	return proto.ExecResultOK, nil
}

func (x *TestExec) Executorcoord_finish_block(ctx context.Context, blockID proto.BlockID) error {
	if db, ok := x.db.(interface{ IsOpenBlock(proto.BlockID) bool }); ok {
		must.Assertf(ctx, db.IsOpenBlock(blockID), "finish: db block not open")
	}
	x.lk.Lock()
	defer x.lk.Unlock()
	must.Assertf(ctx, x.open[blockID] > 0, "exec block not open")
	x.open[blockID] -= 1
	return nil
}

func (x *TestExec) Executorcoord_shutdown(ctx context.Context) error {
	return nil
}

func (x *TestExec) Close() {
}
