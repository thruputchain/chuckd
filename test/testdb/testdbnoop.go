package testdb

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	proto "github.com/groundhogchain/protocol-go"
)

type testDbNoop struct {
}

func NewTestDbNoop() TestDb {
	return &testDbNoop{}
}

func (db *testDbNoop) GetTip() batonapi.Tip {
	return batonapi.Tip{}
}

func (db *testDbNoop) RestoreDb(context.Context) (batonapi.Tip, error) {
	return batonapi.Tip{}, nil
}

func (db *testDbNoop) DbCoord() batonapi.DbCoordConn {
	return db
}

func (db *testDbNoop) AddProposedTx(ctx context.Context, id proto.BlockID, ptx *proto.ProposedTransaction) {
}

func (db *testDbNoop) AddSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction) {
}

func (db *testDbNoop) Dbcoord_null(ctx context.Context) error {
	return nil
}

func (db *testDbNoop) Dbcoord_get_tip(context.Context) (proto.DbTipResult, error) {
	tip := batonapi.Tip{}
	r := proto.DbTipResult{Result: proto.DbResultOK}
	*r.Db_tip() = proto.DbTip{Height: tip.Height, Hash: tip.Hash, Id: tip.ID}
	return r, nil
}

func (db *testDbNoop) Dbcoord_open_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error) {
	return proto.DbResultOK, nil
}

func (db *testDbNoop) Dbcoord_create_snapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) (proto.DbResult, error) {
	return proto.DbResultOK, nil
}

func (db *testDbNoop) Dbcoord_commit_snapshot(ctx context.Context, id proto.BlockID) (proto.DbBlockResult, error) {
	result := proto.DbBlockResult{Result: proto.DbResultOK}
	return result, nil
}

func (db *testDbNoop) Dbcoord_close_snapshot(ctx context.Context, id proto.BlockID) (proto.DbBlockResult, error) {
	result := proto.DbBlockResult{Result: proto.DbResultOK}
	return result, nil
}

func (db *testDbNoop) Dbcoord_abandon_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error) {
	return proto.DbResultOK, nil
}

func (db *testDbNoop) Dbcoord_catchup(context.Context, proto.BlockHeight) (proto.DbCatchupResult, error) {
	panic("not used here")
}

func (db *testDbNoop) Close() {
}
