package testdb

import (
	"context"
	"sync"

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
	tip       proto.DbTip
	open      int
	committed bool
}

func NewTestDbConsistent(tip proto.DbTip) TestDb {
	db := &testDbConsistent{
		m:  xdrutil.NewXdrMarshaler(),
		id: map[proto.BlockID]*testDbSnapshot{},
	}
	db.top = &testDbSnapshot{
		tip:       tip,
		open:      0,
		committed: true,
	}
	db.id[tip.Id] = db.top
	return db
}

func (db *testDbConsistent) GetTip() proto.DbTip {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip
}

func (db *testDbConsistent) RestoreDb(context.Context) (proto.DbTip, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip, nil
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

func (db *testDbConsistent) OpenSnapshot(ctx context.Context, id proto.BlockID) {
	// liblog.Debugf("db open %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()

	snap := db.id[id]
	must.Assertf(ctx, snap != nil, "block not found")
	snap.open += 1
}

func (db *testDbConsistent) CreateSnapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) {
	// liblog.Debugf("db create %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()

	parent, ok := db.id[parentID]
	must.Assertf(ctx, ok, "missing parent block")
	child := &testDbSnapshot{
		tip: proto.DbTip{
			Height: parent.tip.Height + 1,
			Hash:   parent.tip.Hash,
			Id:     id,
		},
		open:      1,
		committed: false,
	}
	must.Assertf(ctx, db.id[id] == nil, "block id already allocated")
	db.id[id] = child
}

func (db *testDbConsistent) CommitSnapshot(ctx context.Context, id proto.BlockID) proto.BlockHash {
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
	return snap.tip.Hash
}

func (db *testDbConsistent) CloseSnapshot(ctx context.Context, id proto.BlockID) proto.BlockHash {
	// liblog.Debugf("db close %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open > 0, "block not open")
	snap.open -= 1
	return snap.tip.Hash
}

func (db *testDbConsistent) AbandonSnapshot(ctx context.Context, id proto.BlockID) {
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
}

func (db *testDbConsistent) GetCatchup(context.Context, proto.BlockHeight) *proto.CatchupBlock {
	panic("not used yet")
}
