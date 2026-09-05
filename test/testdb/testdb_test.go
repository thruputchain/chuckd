package testdb

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func assertTipEqual(t *testing.T, got, want proto.DbTip) {
	t.Helper()
	// XDR unions lazily materialize their active arm during serialization.
	// Compare persisted values, not whether the in-memory arm was allocated.
	gotJSON, err := libxdr.XdrToJson(&got)
	if err != nil {
		t.Fatalf("marshal actual tip: %v", err)
	}
	wantJSON, err := libxdr.XdrToJson(&want)
	if err != nil {
		t.Fatalf("marshal expected tip: %v", err)
	}
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("unexpected tip: got=%s want=%s", gotJSON, wantJSON)
	}
}

func tipAt(height int64, id proto.BlockID) proto.DbTip {
	return proto.DbTip{
		Id: id,
		Value: proto.ConsensusValue{
			Proposal_header: proto.ProposalHeader{
				Block_height: proto.BlockHeight(height),
			},
		},
	}
}

func TestNewTestDbLoadsPersistedTip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	persisted := tipAt(7, 99)
	js, err := libxdr.XdrToJson(&persisted)
	if err != nil {
		t.Fatalf("marshal persisted tip: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dbtip.json"), js, 0o644); err != nil {
		t.Fatalf("write persisted tip: %v", err)
	}

	genesis := tipAt(0, 1)
	db := NewTestDb(ctx, genesis, dir)
	assertTipEqual(t, db.GetTip(), persisted)
}

func TestNewTestDbFallsBackToGenesisTip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	genesis := tipAt(0, 42)

	db := NewTestDb(ctx, genesis, filepath.Join(dir, "missing", "nested"))
	assertTipEqual(t, db.GetTip(), genesis)
}

func TestCommitSnapshotPersistsTip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	genesis := tipAt(0, 1)

	db := NewTestDb(ctx, genesis, dir)
	db.CreateSnapshot(ctx, genesis.Id, 2, proto.KBlockExecutionModeProposal)
	db.CommitSnapshot(ctx, 2, tipAt(1, 2).Value)
	want := db.GetTip()

	reloaded := NewTestDb(ctx, tipAt(0, 777), dir)
	assertTipEqual(t, reloaded.GetTip(), want)
}
