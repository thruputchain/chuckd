package baton

import (
	"context"
	"sync"

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

func (db *TestDb) Dbcoord_null(context.Context) error {
	return nil
}

func (db *TestDb) Dbcoord_open_snapshot(context.Context, proto.BlockID, proto.BlockID) (proto.DbResult, error) {
	XXX
}

func (db *TestDb) Dbcoord_commit_snapshot(context.Context, proto.BlockID) (proto.DbBlockResult, error) {
	XXX
}

func (db *TestDb) Dbcoord_close_snapshot(context.Context, proto.BlockID) (proto.DbBlockResult, error) {
	XXX
}

func (db *TestDb) Dbcoord_abandon_snapshot(context.Context, proto.BlockID) (proto.DbResult, error) {
	XXX
}

func (db *TestDb) Close() error {
	return nil
}
