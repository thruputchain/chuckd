package testdb

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type TestDb struct {
	lk    sync.Mutex
	id    map[proto.BlockID]*testDbSnapshot
	top   *testDbSnapshot
	dbDir string
}

type testDbSnapshot struct {
	// While the snapshot is being built, only Id and the header's
	// Block_height and Storage_hash of tip.Value.Proposal_header are
	// meaningful. On commit, tip.Value is replaced with the committed
	// consensus value.
	tip       proto.DbTip
	hashes    []proto.SignedTransactionHash
	open      int
	committed bool
}

func NewTestDb(ctx context.Context, genesisTip proto.DbTip, dbDir string) *TestDb {
	must.Assert(ctx, genesisTip.Value.Proposal_header.Block_height >= 0)
	must.NoError(ctx, os.MkdirAll(dbDir, 0o755))
	tip := loadTipFromDisk(dbDir, genesisTip)
	db := &TestDb{
		id:    map[proto.BlockID]*testDbSnapshot{},
		dbDir: dbDir,
	}
	db.top = &testDbSnapshot{
		tip:       tip,
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
	header := &snap.tip.Value.Proposal_header
	header.Storage_hash = testutil.BlendPropopsedTx(header.Storage_hash, ptx)
	snap.hashes = append(snap.hashes, protoutil.HashSignedTx(&ptx.Signed_tx))
}

func (db *TestDb) AddSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction) {
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok && snap.open > 0, "block is not open")
	header := &snap.tip.Value.Proposal_header
	header.Storage_hash = testutil.BlendSignedTx(header.Storage_hash, stx)
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

func (db *TestDb) CreateSnapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) {
	// liblog.Debugf("db create %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()

	parent, ok := db.id[parentID]
	must.Assertf(ctx, ok, "missing parent block")
	parentHeader := parent.tip.Value.Proposal_header
	child := &testDbSnapshot{
		tip: proto.DbTip{
			Id: id,
			Value: proto.ConsensusValue{
				Proposal_header: proto.ProposalHeader{
					Network_id:   parentHeader.Network_id,
					Block_height: parentHeader.Block_height + 1,
					Storage_hash: parentHeader.Storage_hash,
				},
			},
		},
		open:      1,
		committed: false,
	}
	must.Assertf(ctx, db.id[id] == nil, "block id already allocated")
	db.id[id] = child
}

func (db *TestDb) CommitSnapshot(ctx context.Context, id proto.BlockID, value proto.ConsensusValue) []proto.SignedTransactionHash {
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
	return snap.hashes
}

// CloseSnapshot finalizes the snapshot's storage hash and returns the partial
// proposal header (block height, storage hash, network id). The caller fills
// in the remaining header fields (timestamp, txs hash, events hash).
func (db *TestDb) CloseSnapshot(ctx context.Context, id proto.BlockID) proto.ProposalHeader {
	// liblog.Debugf("db close %v", form.Display(id))
	db.lk.Lock()
	defer db.lk.Unlock()
	snap, ok := db.id[id]
	must.Assertf(ctx, ok, "unknown block")
	must.Assertf(ctx, snap.open > 0, "block not open")
	snap.open -= 1
	header := &snap.tip.Value.Proposal_header
	header.Storage_hash = testutil.UniqueStorageHash(header.Block_height, header.Storage_hash)
	return *header
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
