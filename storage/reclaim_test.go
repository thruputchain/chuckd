package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

func TestReclaimCatchupBlocksRemovesOldest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		Freq:             time.Hour,
		MaxCatchupBlocks: 2,
		MaxSCPSlots:      10,
	}

	st := NewStorage(ctx, cfg)
	st.Start(ctx, dir)

	heights := []int64{101, 102, 103, 104}
	for i, h := range heights {
		cv := newConsensusValue(time.Unix(1800000000+int64(i), 0).UTC(), h)
		cb := proto.CatchupBlock{
			Consensus_value:     cv,
			Signed_transactions: nil,
		}
		st.PersistCatchupBlock(ctx, cb)
	}

	files, err := ListCatchupFiles(st.CatchupBlocksDirpath())
	if err != nil {
		t.Fatalf("ListCatchupFiles (before reclaim) returned error: %v", err)
	}
	if len(files) != len(heights) {
		t.Fatalf("expected %d catchup files before reclaim, got %d", len(heights), len(files))
	}

	st.reclaimCatchupBlocks(ctx)

	files, err = ListCatchupFiles(st.CatchupBlocksDirpath())
	if err != nil {
		t.Fatalf("ListCatchupFiles (after reclaim) returned error: %v", err)
	}

	if len(files) != cfg.MaxCatchupBlocks {
		t.Fatalf("expected %d catchup files after reclaim, got %d", cfg.MaxCatchupBlocks, len(files))
	}

	want := []int64{heights[2], heights[3]}
	for i, f := range files {
		base := filepath.Base(f)
		heightStr := strings.TrimSuffix(strings.TrimPrefix(base, "catchup-"), ".xdr")
		height, err := strconv.ParseInt(heightStr, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse catchup filename %q: %v", base, err)
		}
		if height != want[i] {
			t.Fatalf("catchup file %d height mismatch: got %d, want %d", i, height, want[i])
		}
	}

	if found := st.FindCatchupBlock(ctx, proto.BlockHeight(heights[0])); found != nil {
		t.Fatalf("expected catchup block %d to be reclaimed, got some", heights[0])
	}
	if found := st.FindCatchupBlock(ctx, proto.BlockHeight(heights[2])); found == nil {
		t.Fatalf("expected catchup block %d to remain readable, got none", heights[2])
	}
}

func TestReclaimSCPEnvelopesRemovesOldestSlots(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		Freq:             time.Hour,
		MaxCatchupBlocks: 10,
		MaxSCPSlots:      2,
	}

	st := NewStorage(ctx, cfg)
	st.Start(ctx, dir)

	slots := []int64{5, 6, 7}
	for i, slot := range slots {
		env := newEnvelope(slot, byte(0x20+i))
		st.PersistSCPEnvelope(ctx, env)
	}

	slotdirs, err := ListSCPSlotDirs(st.SCPEnvelopesDirpath())
	if err != nil {
		t.Fatalf("ListSCPSlotDirs (before reclaim) returned error: %v", err)
	}
	if len(slotdirs) != len(slots) {
		t.Fatalf("expected %d slot directories before reclaim, got %d", len(slots), len(slotdirs))
	}

	st.reclaimSCPEnvelopes(ctx)

	slotdirs, err = ListSCPSlotDirs(st.SCPEnvelopesDirpath())
	if err != nil {
		t.Fatalf("ListSCPSlotDirs (after reclaim) returned error: %v", err)
	}
	if len(slotdirs) != cfg.MaxSCPSlots {
		t.Fatalf("expected %d slot directories after reclaim, got %d", cfg.MaxSCPSlots, len(slotdirs))
	}

	wantSlots := []int64{slots[1], slots[2]}
	for i, dirpath := range slotdirs {
		base := filepath.Base(dirpath)
		slotNum, err := strconv.ParseInt(base, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse slot directory %q: %v", base, err)
		}
		if slotNum != wantSlots[i] {
			t.Fatalf("slot directory %d mismatch: got %d, want %d", i, slotNum, wantSlots[i])
		}
	}

	removedPath := st.SCPEnvelopeSlotpath(proto.SCPSlotIndex(slots[0]))
	if _, err := os.Stat(removedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected slot directory %s to be reclaimed, got error %v", removedPath, err)
	}
}
