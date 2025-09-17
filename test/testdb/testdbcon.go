package testdb

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type testDbConsistent struct {
	m   xdrutil.XdrMarshaler
	lk  sync.Mutex
	id  map[proto.BlockID]*testDbSnapshot
	top *testDbSnapshot
}

type testDbSnapshot struct {
	tip       batonapi.Tip
	open      int
	committed bool
}

func NewTestDbConsistent(tip batonapi.Tip) TestDb {
	db := &testDbConsistent{
		m:  xdrutil.NewXdrMarshaler(),
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

func (db *testDbConsistent) GetTip() batonapi.Tip {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip
}

func (db *testDbConsistent) RestoreDb(context.Context) (batonapi.Tip, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip, nil
}

func (db *testDbConsistent) DbCoord() batonapi.DbCoordConn {
	return db
}

// AssertOpen is called by TestExec
func (db *testDbConsistent) IsOpenBlock(id proto.BlockID) bool {
	db.lk.Lock()
	defer db.lk.Unlock()
	block, ok := db.id[id]
	return ok && block.open > 0
}

func (db *testDbConsistent) AddProposedTx(ctx context.Context, id proto.BlockID, ptx *proto.ProposedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open > 0, "block is not open")
	snap.tip.Hash = testutil.BlendPropopsedTx(db.m, snap.tip.Hash, ptx)
}

func (db *testDbConsistent) AddSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open > 0, "block is not open")
	snap.tip.Hash = testutil.BlendSignedTx(db.m, snap.tip.Hash, stx)
}

func (db *testDbConsistent) Dbcoord_null(ctx context.Context) error {
	return nil
}

func (db *testDbConsistent) Dbcoord_get_tip(context.Context) (proto.DbTipResult, error) {
	tip := db.GetTip()
	r := proto.DbTipResult{Result: proto.DbResultOK}
	*r.Db_tip() = proto.DbTip{Height: tip.Height, Hash: tip.Hash, Id: tip.ID}
	return r, nil
}

func (db *testDbConsistent) Dbcoord_open_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error) {
	// liblog.Debugf("db open %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()

	snap := db.id[id]
	must.Assertf(ctx, snap != nil, "block not found")
	snap.open += 1

	return proto.DbResultOK, nil
}

func (db *testDbConsistent) Dbcoord_create_snapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) (proto.DbResult, error) {
	// liblog.Debugf("db create %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()

	parent, ok := db.id[parentID]
	must.Assertf(ctx, ok, "missing parent block")
	child := &testDbSnapshot{
		tip: batonapi.Tip{
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

func (db *testDbConsistent) Dbcoord_commit_snapshot(ctx context.Context, id proto.BlockID) (proto.DbBlockResult, error) {
	// NOTE: opening a block before commit is no longer the expected db behavior in the baton,
	// so we simulate it here to conform to what the testdb expects
	r, err := db.Dbcoord_open_snapshot(ctx, id)
	must.NoError(ctx, err)
	must.Assert(ctx, r == proto.DbResultOK)

	// liblog.Debugf("db commit %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open > 0, "block not open")
	must.Assertf(ctx, !snap.committed, "block already committed")
	snap.open -= 1
	snap.committed = true
	db.top = snap
	result := proto.DbBlockResult{Result: proto.DbResultOK}
	*result.Block_hash() = snap.tip.Hash
	return result, nil
}

func (db *testDbConsistent) Dbcoord_close_snapshot(ctx context.Context, id proto.BlockID) (proto.DbBlockResult, error) {
	// liblog.Debugf("db close %v", form.Display(id))
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

func (db *testDbConsistent) Dbcoord_abandon_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error) {
	// liblog.Debugf("db abandon %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open > 0, "block is not open")
	must.Assertf(ctx, !snap.committed, "abandoning committed block")
	snap.open -= 1
	if snap.open == 0 {
		delete(db.id, id)
	}
	return proto.DbResultOK, nil
}

func (db *testDbConsistent) Dbcoord_catchup(context.Context, proto.BlockHeight) (proto.DbCatchupResult, error) {
	panic("not used here")
}

func (db *testDbConsistent) Dbcoord_get_hog_ages(context.Context, proto.ContractAddressBatch) (proto.HogAgeResult, error) {
	panic("not used")
}

func (db *testDbConsistent) Dbcoord_write_hog_ages(context.Context, proto.HogAgeBatch) (proto.DbResult, error) {
	panic("not used")
}

func (db *testDbConsistent) Dbcoord_load_genesis_block(context.Context, proto.GenesisBlock) (proto.DbResult, error) {
	panic("not used")
}

func (db *testDbConsistent) Dbcoord_shard_restore(ctx context.Context, a1 proto.BlockHeight) (r proto.DbShardResult, _ error) {
	return proto.ShardOK, nil
}

func (db *testDbConsistent) Close() {
}
