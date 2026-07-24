package testdb

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

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
	if got := db.GetTip(); !reflect.DeepEqual(got, persisted) {
		t.Fatalf("unexpected loaded tip: got=%+v want=%+v", got, persisted)
	}
}

func TestNewTestDbFallsBackToGenesisTip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	genesis := tipAt(0, 42)

	db := NewTestDb(ctx, genesis, filepath.Join(dir, "missing", "nested"))
	if got := db.GetTip(); !reflect.DeepEqual(got, genesis) {
		t.Fatalf("unexpected genesis fallback: got=%+v want=%+v", got, genesis)
	}
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
	if got := reloaded.GetTip(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected reloaded tip: got=%+v want=%+v", got, want)
	}
}
