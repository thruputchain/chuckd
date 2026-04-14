package testdb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func TestNewTestDbLoadsPersistedTip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	persisted := proto.DbTip{
		Height: 7,
		Id:     99,
	}
	js, err := libxdr.XdrToJson(&persisted)
	if err != nil {
		t.Fatalf("marshal persisted tip: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dbtip.json"), js, 0o644); err != nil {
		t.Fatalf("write persisted tip: %v", err)
	}

	genesis := proto.DbTip{Height: 0, Id: 1}
	db := NewTestDb(ctx, genesis, dir)
	if got := db.GetTip(); got != persisted {
		t.Fatalf("unexpected loaded tip: got=%+v want=%+v", got, persisted)
	}
}

func TestNewTestDbFallsBackToGenesisTip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	genesis := proto.DbTip{Height: 0, Id: 42}

	db := NewTestDb(ctx, genesis, filepath.Join(dir, "missing", "nested"))
	if got := db.GetTip(); got != genesis {
		t.Fatalf("unexpected genesis fallback: got=%+v want=%+v", got, genesis)
	}
}

func TestCommitSnapshotPersistsTip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	genesis := proto.DbTip{Height: 0, Id: 1}

	db := NewTestDb(ctx, genesis, dir)
	db.CreateSnapshot(ctx, genesis.Id, 2, proto.KBlockExecutionModeProposal)
	db.CommitSnapshot(ctx, 2)
	want := db.GetTip()

	reloaded := NewTestDb(ctx, proto.DbTip{Height: 0, Id: 777}, dir)
	if got := reloaded.GetTip(); got != want {
		t.Fatalf("unexpected reloaded tip: got=%+v want=%+v", got, want)
	}
}
