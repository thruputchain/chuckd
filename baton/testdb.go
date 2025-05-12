package baton

import (
	"context"
	"crypto/sha256"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type TestDb struct {
	lk  sync.Mutex
	id  map[proto.BlockID]*testDbSnapshot
	tip *testDbSnapshot
}

type testDbSnapshot struct {
	height    proto.BlockHeight
	hash      proto.BlockHash
	id        proto.BlockID
	open      int
	committed bool
}

func NewTestDb(height proto.BlockHeight, hash proto.BlockHash, id proto.BlockID) *TestDb {
	db := &TestDb{
		id: map[proto.BlockID]*testDbSnapshot{},
	}
	db.tip = &testDbSnapshot{
		height:    height,
		hash:      hash,
		id:        id,
		open:      0,
		committed: true,
	}
	db.id[id] = db.tip
	return db
}

func (db *TestDb) RestoreDb(context.Context) (proto.BlockHeight, proto.BlockHash, proto.BlockID, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.tip.height, db.tip.hash, db.tip.id, nil
}

func (db *TestDb) DbCoord() DbCoordConn {
	return db
}

// assertOpen is called by TestExec
func (db *TestDb) assertOpen(ctx context.Context, id proto.BlockID) {
	db.lk.Lock()
	defer db.lk.Unlock()
	block, ok := db.id[id]
	must.Assertf(ctx, ok && block.open > 0, "block is not open")
}

// mutate is called by TestExec
func (db *TestDb) mutate(ctx context.Context, id proto.BlockID, mut []byte) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open > 0, "block is not open")
	snap.hash = sha256.Sum256(append(snap.hash[:], mut...))
}

func (db *TestDb) Dbcoord_null(ctx context.Context) error {
	return nil
}

func (db *TestDb) Dbcoord_open_snapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID) (proto.DbResult, error) {
	db.lk.Lock()
	defer db.lk.Unlock()

	parent, ok := db.id[parentID]
	must.Assertf(ctx, ok, "missing parent block")
	child := &testDbSnapshot{
		height:    parent.height + 1,
		hash:      proto.BlockHash{}, // 0
		id:        id,
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
	*result.Block_hash() = snap.hash
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
	*result.Block_hash() = snap.hash
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

func (db *TestDb) Close() error {
	return nil
}
