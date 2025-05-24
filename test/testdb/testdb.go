package testdb

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/test/testutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestDb struct {
	lk  sync.Mutex
	id  map[proto.BlockID]*testDbSnapshot
	top *testDbSnapshot
}

type testDbSnapshot struct {
	tip       sdk.Tip
	open      int
	committed bool
}

func NewTestDb(tip sdk.Tip) *TestDb {
	db := &TestDb{
		id: map[proto.BlockID]*testDbSnapshot{},
	}
	db.top = &testDbSnapshot{
		tip:       tip,
		open:      0,
		committed: true,
	}
	db.id[tip.ID] = db.top
	return db
}

func (db *TestDb) GetTip() sdk.Tip {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip
}

func (db *TestDb) RestoreDb(context.Context) (sdk.Tip, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip, nil
}

func (db *TestDb) DbCoord() sdk.DbCoordConn {
	return db
}

// AssertOpen is called by TestExec
func (db *TestDb) AssertOpen(ctx context.Context, id proto.BlockID) {
	db.lk.Lock()
	defer db.lk.Unlock()
	block, ok := db.id[id]
	must.Assertf(ctx, ok && block.open > 0, "block is not open")
}

func (db *TestDb) AddProposedTx(ctx context.Context, id proto.BlockID, ptx *proto.ProposedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open > 0, "block is not open")
	snap.tip.Hash = testutil.BlendPropopsedTx(snap.tip.Hash, ptx)
}

func (db *TestDb) AddSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open > 0, "block is not open")
	snap.tip.Hash = testutil.BlendSignedTx(snap.tip.Hash, stx)
}

func (db *TestDb) Dbcoord_null(ctx context.Context) error {
	return nil
}

func (db *TestDb) Dbcoord_get_tip(context.Context) (proto.DbTipResult, error) {
	tip := db.GetTip()
	r := proto.DbTipResult{Result: proto.DbResultOK}
	*r.Db_tip() = proto.DbTip{Height: tip.Height, Hash: tip.Hash, Id: tip.ID}
	return r, nil
}

func (db *TestDb) Dbcoord_open_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()

	snap := db.id[id]
	must.Assertf(ctx, snap != nil, "block not found")
	snap.open += 1

	return proto.DbResultOK, nil
}

func (db *TestDb) Dbcoord_create_snapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) (proto.DbResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()

	parent, ok := db.id[parentID]
	must.Assertf(ctx, ok, "missing parent block")
	child := &testDbSnapshot{
		tip: sdk.Tip{
			Height: parent.tip.Height + 1,
			Hash:   parent.tip.Hash,
			ID:     id,
		},
		open:      1,
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
	must.Assertf(ctx, snap.open > 0, "block not open")
	must.Assertf(ctx, !snap.committed, "block already committed")
	snap.open -= 1
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
	must.Assertf(ctx, snap.open > 0, "block not open")
	snap.open -= 1
	result := proto.DbBlockResult{Result: proto.DbResultOK}
	*result.Block_hash() = snap.tip.Hash
	return result, nil
}

func (db *TestDb) Dbcoord_abandon_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open > 0, "block is not open")
	must.Assertf(ctx, !snap.committed, "abandoning committed block")
	snap.open -= 1
	delete(db.id, id)
	return proto.DbResultOK, nil
}

func (db *TestDb) Dbcoord_catchup(context.Context, proto.BlockHeight) (proto.DbCatchupResult, error) {
	panic("not used here")
}

func (db *TestDb) Close() {
}
