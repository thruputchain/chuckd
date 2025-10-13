package testdb

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type testDbNoop struct {
}

func NewTestDbNoop() TestDb {
	return &testDbNoop{}
}

func (db *testDbNoop) GetTip() proto.DbTip {
	return proto.DbTip{}
}

func (db *testDbNoop) RestoreDb(context.Context) (proto.DbTip, error) {
	return proto.DbTip{}, nil
}

func (db *testDbNoop) AddProposedTx(ctx context.Context, id proto.BlockID, ptx *proto.ProposedTransaction) {
}

func (db *testDbNoop) AddSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction) {
}

func (db *testDbNoop) OpenSnapshot(ctx context.Context, id proto.BlockID) {
}

func (db *testDbNoop) CreateSnapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) {
}

func (db *testDbNoop) CommitSnapshot(ctx context.Context, id proto.BlockID) proto.BlockHash {
	return proto.BlockHash{}
}

func (db *testDbNoop) CloseSnapshot(ctx context.Context, id proto.BlockID) proto.BlockHash {
	return proto.BlockHash{}
}

func (db *testDbNoop) AbandonSnapshot(ctx context.Context, id proto.BlockID) {
}

func (db *testDbNoop) GetCatchup(context.Context, proto.BlockHeight) *proto.CatchupBlock {
	panic("not used here")
}
