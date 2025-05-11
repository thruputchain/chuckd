package baton

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type TestExec struct {
	db *TestDb
}

func NewTestExec(db *TestDb) *TestExec {
	return &TestExec{db: db}
}

func (x *TestExec) Executorcoord_null(context.Context) error {
	return nil
}

func (x *TestExec) Executorcoord_propose_transaction(ctx context.Context, blockID proto.BlockID, stx proto.SignedTransaction) (proto.ExecProposeResult, error) {
	XXX
}

func (x *TestExec) Executorcoord_validate_transaction(ctx context.Context, blockID proto.BlockID, ptx proto.ProposedTransaction) (proto.ExecResult, error) {
	XXX
}

func (x *TestExec) Executorcoord_open_proposal_block(ctx context.Context, blockID proto.BlockID) (proto.ExecResult, error) {
	XXX
}

func (x *TestExec) Executorcoord_open_validation_block(ctx context.Context, blockID proto.BlockID) (proto.ExecResult, error) {
	XXX
}

func (x *TestExec) Executorcoord_finish_block(ctx context.Context, blockID proto.BlockID) error {
	XXX
}

func (x *TestExec) Executorcoord_shutdown(ctx context.Context) error {
	return nil
}

func (db *TestExec) Close() error {
	return nil
}
