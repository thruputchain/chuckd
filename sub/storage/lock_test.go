package storage

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestDiskStorageLockHeldAfterGC verifies that a second storage lock cannot be acquired.
func TestDiskStorageLockHeldAfterGC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir := t.TempDir()
	cfg := Config{
		Freq:             time.Hour,
		MaxCatchupBlocks: 10,
		MaxSCPSlots:      10,
	}

	st1 := NewStorage(ctx, cfg)
	st1.Start(ctx, dir)

	if !diskStorageStartPanics(ctx, cfg, dir) {
		t.Fatalf("control check failed: second storage instance acquired the lock before GC")
	}

	runtime.GC()
	runtime.GC()

	if !diskStorageStartPanics(ctx, cfg, dir) {
		t.Fatalf("control check failed: second storage instance acquired the lock after GC")
	}

	runtime.KeepAlive(st1)
}

func diskStorageStartPanics(ctx context.Context, cfg Config, dir string) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()

	st := NewStorage(ctx, cfg)
	st.Start(ctx, dir)
	return false
}
