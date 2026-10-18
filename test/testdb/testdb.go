package testdb

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/thruputchain/chuckd/test/testutil"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type TestDb struct {
	lk    sync.Mutex
	id    map[proto.BlockID]*testDbSnapshot
	top   *testDbSnapshot
	dbDir string
}

type testDbSnapshot struct {
	tip       proto.DbTip // id and, once committed, the consensus value that committed the block
	height    proto.BlockHeight
	storage   proto.StorageShortHash // rolling storage hash of applied txs
	hashes    []proto.TransactionHash
	open      int
	committed bool
}

func NewTestDb(ctx context.Context, genesisTip proto.DbTip, dbDir string) *TestDb {
	must.Assert(ctx, protoutil.ValueHeight(&genesisTip.Value) >= 0)
	must.NoError(ctx, os.MkdirAll(dbDir, 0o755))
	tip := loadTipFromDisk(dbDir, genesisTip)
	db := &TestDb{
		id:    map[proto.BlockID]*testDbSnapshot{},
		dbDir: dbDir,
	}
	db.top = &testDbSnapshot{
		tip:       tip,
		height:    protoutil.ValueHeight(&tip.Value),
		storage:   tip.Value.Proposal_header.Storage_hash,
		open:      0,
		committed: true,
	}
	db.id[tip.Id] = db.top
	return db
}

func (db *TestDb) GetTip() proto.DbTip {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip
}

func (db *TestDb) RestoreDb(context.Context) (proto.DbTip, error) {
	db.lk.Lock()
	defer db.lk.Unlock()
	return db.top.tip, nil
}

// AssertOpen is called by TestExec
func (db *TestDb) IsOpenBlock(id proto.BlockID) bool {
	db.lk.Lock()
	defer db.lk.Unlock()
	block, ok := db.id[id]
	return ok && block.open > 0
}

func (db *TestDb) AddProposedTx(ctx context.Context, id proto.BlockID, ptx *proto.ProposedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open > 0, "block is not open")
	snap.storage = testutil.BlendPropopsedTx(snap.storage, ptx)
	snap.hashes = append(snap.hashes, protoutil.HashSignedTx(&ptx.Signed_tx))
}

func (db *TestDb) AddSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open > 0, "block is not open")
	snap.storage = testutil.BlendSignedTx(snap.storage, stx)
	snap.hashes = append(snap.hashes, protoutil.HashSignedTx(stx))
}

func (db *TestDb) OpenSnapshot(ctx context.Context, id proto.BlockID) {
	// liblog.Debugf("db open %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()

	snap := db.id[id]
	must.Assertf(ctx, snap != nil, "block not found")
	snap.open += 1
}

// CreateSnapshot creates a child snapshot of the parent block and returns the child's height.
func (db *TestDb) CreateSnapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) proto.BlockHeight {
	// liblog.Debugf("db create %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()

	parent, ok := db.id[parentID]
	must.Assertf(ctx, ok, "missing parent block")
	child := &testDbSnapshot{
		tip:       proto.DbTip{Id: id},
		height:    parent.height + 1,
		storage:   parent.storage,
		open:      1,
		committed: false,
	}
	must.Assertf(ctx, db.id[id] == nil, "block id already allocated")
	db.id[id] = child
	return child.height
}

// CommitSnapshot commits the block and associates it with the consensus value that confirmed it.
func (db *TestDb) CommitSnapshot(ctx context.Context, id proto.BlockID, value proto.ConsensusValue) (proto.StorageShortHash, []proto.TransactionHash) {
	// liblog.Debugf("db commit %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open > 0, "block not open")
	must.Assertf(ctx, !snap.committed, "block already committed")
	snap.open -= 1
	snap.committed = true
	snap.tip.Value = value
	db.top = snap
	persistTipToDisk(ctx, db.dbDir, db.top.tip)
	return snap.storage, snap.hashes
}

func (db *TestDb) CloseSnapshot(ctx context.Context, id proto.BlockID) proto.StorageShortHash {
	// liblog.Debugf("db close %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open > 0, "block not open")
	snap.open -= 1
	snap.storage = testutil.UniqueStorageHash(snap.height, snap.storage)
	return snap.storage
}

func (db *TestDb) AbandonSnapshot(ctx context.Context, id proto.BlockID) {
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

func (db *TestDb) GetCatchup(context.Context, proto.BlockHeight) *proto.CatchupBlock {
	panic("not used yet")
}

func dbTipPath(dir string) string {
	return filepath.Join(dir, "dbtip.json")
}

func loadTipFromDisk(dir string, genesisTip proto.DbTip) proto.DbTip {
	data, err := os.ReadFile(dbTipPath(dir))
	if err != nil {
		return genesisTip
	}
	var tip proto.DbTip
	if err := libxdr.JsonToXdr(&tip, data); err != nil {
		return genesisTip
	}
	return tip
}

func persistTipToDisk(ctx context.Context, dir string, tip proto.DbTip) {
	full := dbTipPath(dir)
	staging, err := os.CreateTemp(dir, "dbtip-*.json")
	must.NoError(ctx, err)
	defer func() {
		_ = os.Remove(staging.Name())
	}()
	js, err := libxdr.XdrToJson(&tip)
	must.NoError(ctx, err)
	_, err = staging.Write(js)
	must.NoError(ctx, err)
	must.NoError(ctx, staging.Close())
	must.NoError(ctx, os.Rename(staging.Name(), full))
}
