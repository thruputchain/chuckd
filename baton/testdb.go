package baton

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type TestDb struct {
	lk  sync.Mutex
	id  map[proto.BlockID]*testDbSnapshot
	top *testDbSnapshot
}

type testDbSnapshot struct {
	tip       Tip
	open      bool
	committed bool
}

func NewTestDb(tip Tip) *TestDb {
	db := &TestDb{
		id: map[proto.BlockID]*testDbSnapshot{},
	}
	db.top = &testDbSnapshot{
		tip:       tip,
		open:      false,
		committed: true,
	}
	db.id[tip.ID] = db.top
	return db
}

func (db *TestDb) GetTip() Tip {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip
}

func (db *TestDb) RestoreDb(context.Context) (Tip, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip, nil
}

func (db *TestDb) DbCoord() DbCoordConn {
	return db
}

// assertOpen is called by TestExec
func (db *TestDb) assertOpen(ctx context.Context, id proto.BlockID) {
	db.lk.Lock()
	defer db.lk.Unlock()
	block, ok := db.id[id]
	must.Assertf(ctx, ok && block.open, "block is not open")
}

func (db *TestDb) addProposedTx(ctx context.Context, id proto.BlockID, ptx *proto.ProposedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open, "block is not open")
	snap.tip.Hash = blendPropopsedTx(snap.tip.Hash, ptx)
}

func (db *TestDb) addSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open, "block is not open")
	snap.tip.Hash = blendSignedTx(snap.tip.Hash, stx)
}

func (db *TestDb) Dbcoord_null(ctx context.Context) error {
	return nil
}

func (db *TestDb) Dbcoord_get_tip(context.Context) (proto.DbTipResult, error) {
	panic("not used")
}

func (db *TestDb) Dbcoord_open_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()

	snap := db.id[id]
	must.Assertf(ctx, snap != nil, "block not found")
	must.Assertf(ctx, !snap.open, "block already open")
	snap.open = true

	return proto.DbResultOK, nil
}

func (db *TestDb) Dbcoord_create_snapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) (proto.DbResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()

	parent, ok := db.id[parentID]
	must.Assertf(ctx, ok, "missing parent block")
	child := &testDbSnapshot{
		tip: Tip{
			Height: parent.tip.Height + 1,
			Hash:   parent.tip.Hash,
			ID:     id,
		},
		open:      true,
		committed: false,
	}
	must.Assertf(ctx, db.id[id] == nil, "block id already allocated")
	db.id[id] = child
	return proto.DbResultOK, nil
}

func (db *TestDb) Dbcoord_commit_snapshot(ctx context.Context, id proto.BlockID) (proto.DbBlockResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open, "block not open")
	must.Assertf(ctx, !snap.committed, "block already committed")
	snap.open = false
	snap.committed = true
	result := proto.DbBlockResult{Result: proto.DbResultOK}
	*result.Block_hash() = snap.tip.Hash
	return result, nil
}

func (db *TestDb) Dbcoord_close_snapshot(ctx context.Context, id proto.BlockID) (proto.DbBlockResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open, "block not open")
	snap.open = false
	result := proto.DbBlockResult{Result: proto.DbResultOK}
	*result.Block_hash() = snap.tip.Hash
	return result, nil
}

func (db *TestDb) Dbcoord_abandon_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open, "block is not open")
	must.Assertf(ctx, !snap.committed, "abandoning committed block")
	snap.open = false
	delete(db.id, id)
	return proto.DbResultOK, nil
}

func (db *TestDb) Dbcoord_catchup(context.Context, proto.BlockHeight) (proto.DbCatchupResult, error) {
	panic("not used here")
}

func (db *TestDb) Close() {
}
