package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func TestDiskStorageCatchupBlockRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		Freq:             time.Hour,
		MaxCatchupBlocks: 10,
		MaxSCPSlots:      10,
	}

	st := NewStorage(ctx, dir, cfg)

	cv := newConsensusValue(time.Unix(1700000000, 0).UTC(), 42)
	cb := proto.CatchupBlock{
		Consensus_value:     cv,
		Signed_transactions: nil,
	}

	st.AddCatchupBlock(ctx, cb)

	got, err := st.FindCatchupBlock(ctx, cv.Nomination.Block_height)
	if err != nil {
		t.Fatalf("FindCatchupBlock returned error: %v", err)
	}

	if !reflect.DeepEqual(got, cb) {
		t.Fatalf("FindCatchupBlock mismatch:\n got: %+v\nwant: %+v", got, cb)
	}

	_, err = st.FindCatchupBlock(ctx, cv.Nomination.Block_height+1)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist for missing height, got %v", err)
	}
}

func TestFindLatestConsensusValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		Freq:             time.Hour,
		MaxCatchupBlocks: 10,
		MaxSCPSlots:      10,
	}

	st := NewStorage(ctx, dir, cfg)

	base := time.Date(2024, 10, 18, 12, 0, 0, 0, time.UTC)
	vals := []proto.ConsensusValue{
		newConsensusValue(base, 88),
		newConsensusValue(base.Add(3*time.Minute), 89),
		newConsensusValue(base.Add(6*time.Minute), 90),
	}

	for _, cv := range vals {
		st.AddConsensusValue(ctx, cv)
	}

	// Drop an unrelated file that should be ignored by the walker.
	invalidDir := filepath.Dir(st.ConsensusValueFilepath(
		protoutil.TimeFromTimepoint(vals[0].Timestamp),
		vals[0].Nomination.Block_height,
	))
	if err := os.WriteFile(filepath.Join(invalidDir, "not-a-consensus.xdr"), []byte("junk"), 0o644); err != nil {
		t.Fatalf("write invalid file: %v", err)
	}

	latest, err := st.FindLatestConsensusValues(ctx, 2)
	if err != nil {
		t.Fatalf("FindLatestConsensusValues returned error: %v", err)
	}

	if len(latest) != 2 {
		t.Fatalf("expected 2 consensus values, got %d", len(latest))
	}

	wantHeights := []proto.BlockHeight{
		vals[2].Nomination.Block_height,
		vals[1].Nomination.Block_height,
	}
	for i, cv := range latest {
		if cv.Nomination.Block_height != wantHeights[i] {
			t.Fatalf("consensus value %d height mismatch: got %d, want %d", i, cv.Nomination.Block_height, wantHeights[i])
		}
		if cv.Nomination.Block_hash != vals[2-i].Nomination.Block_hash {
			t.Fatalf("consensus value %d hash mismatch", i)
		}
	}

	all, err := st.FindLatestConsensusValues(ctx, 10)
	if err != nil {
		t.Fatalf("FindLatestConsensusValues (all) returned error: %v", err)
	}
	if len(all) != len(vals) {
		t.Fatalf("expected %d consensus values, got %d", len(vals), len(all))
	}
}

func TestFindSCPEnvelopesSinceSlot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		Freq:             time.Hour,
		MaxCatchupBlocks: 10,
		MaxSCPSlots:      10,
	}

	st := NewStorage(ctx, dir, cfg)

	envs := []proto.SCPEnvelope{
		newEnvelope(3, 0x01),
		newEnvelope(4, 0x02),
		newEnvelope(4, 0x03),
		newEnvelope(6, 0x04),
	}

	for _, env := range envs {
		st.AddSCPEnvelope(ctx, env)
	}

	got, err := st.FindSCPEnvelopesSinceSlot(ctx, proto.SCPSlotIndex(3))
	if err != nil {
		t.Fatalf("FindSCPEnvelopesSinceSlot returned error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 envelopes (slots 3 and 4), got %d", len(got))
	}

	if got[0].Statement.Slot_index != proto.SCPSlotIndex(3) {
		t.Fatalf("first envelope slot mismatch: got %d, want 3", got[0].Statement.Slot_index)
	}
	if got[1].Statement.Slot_index != proto.SCPSlotIndex(4) || got[2].Statement.Slot_index != proto.SCPSlotIndex(4) {
		t.Fatalf("expected envelopes for slot 4 after first gap, got slots %d and %d", got[1].Statement.Slot_index, got[2].Statement.Slot_index)
	}

	afterGap, err := st.FindSCPEnvelopesSinceSlot(ctx, proto.SCPSlotIndex(6))
	if err != nil {
		t.Fatalf("FindSCPEnvelopesSinceSlot (slot 6) returned error: %v", err)
	}
	if len(afterGap) != 1 || afterGap[0].Statement.Slot_index != proto.SCPSlotIndex(6) {
		t.Fatalf("expected single envelope at slot 6, got %+v", afterGap)
	}

	none, err := st.FindSCPEnvelopesSinceSlot(ctx, proto.SCPSlotIndex(10))
	if err != nil {
		t.Fatalf("FindSCPEnvelopesSinceSlot (slot 10) returned error: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no envelopes starting from slot 10, got %d", len(none))
	}
}

func newConsensusValue(ts time.Time, height int64) proto.ConsensusValue {
	return proto.ConsensusValue{
		Timestamp: protoutil.TimepointFromTime(ts),
		Nomination: proto.Nomination{
			Block_height:    proto.BlockHeight(height),
			Block_hash:      newBlockHash(byte(height)),
			Prev_block_hash: newBlockHash(byte(height - 1)),
		},
	}
}

func newEnvelope(slot int64, marker byte) proto.SCPEnvelope {
	env := proto.SCPEnvelope{
		Statement: proto.SCPStatement{
			Slot_index: proto.SCPSlotIndex(slot),
			Pledges: proto.XdrAnon_SCPStatement_Pledges{
				Stmt_type: proto.SCP_ST_NOMINATE,
			},
		},
		Signature: proto.Signature{
			Scheme: proto.Ed25519,
		},
	}
	env.Statement.Pledges.Nominate()
	sig := env.Signature.Ed25519()
	sig[0] = marker
	return env
}

func newBlockHash(fill byte) proto.BlockHash {
	var h proto.BlockHash
	for i := range h {
		h[i] = fill
		fill++
	}
	return h
}
