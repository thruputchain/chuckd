package storage

import (
	"context"
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

	st := NewStorage(ctx, cfg)
	st.Start(ctx, dir)

	cv := newConsensusValue(time.Unix(1700000000, 0).UTC(), 42)
	cb := proto.CatchupBlock{
		Consensus_value:     cv,
		Signed_transactions: nil,
	}

	st.PersistCatchupBlock(ctx, cb)

	got := st.FindCatchupBlock(ctx, cv.Nomination.Block_height)
	if got == nil {
		t.Fatalf("FindCatchupBlock returned no result")
	}

	if !reflect.DeepEqual(*got, cb) {
		t.Fatalf("FindCatchupBlock mismatch:\n got: %+v\nwant: %+v", got, cb)
	}

	found := st.FindCatchupBlock(ctx, cv.Nomination.Block_height+1)
	if found != nil {
		t.Fatalf("expected missing block, got some")
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

	st := NewStorage(ctx, cfg)
	st.Start(ctx, dir)

	envs := []proto.SCPEnvelope{
		newEnvelope(3, 0x01),
		newEnvelope(4, 0x02),
		newEnvelope(4, 0x03),
		newEnvelope(6, 0x04),
	}

	for _, env := range envs {
		st.PersistSCPEnvelope(ctx, env)
	}

	got := st.FindSCPEnvelopesSinceSlot(ctx, proto.SCPSlotIndex(3))

	if len(got) != 3 {
		t.Fatalf("expected 3 envelopes (slots 3 and 4), got %d", len(got))
	}

	if got[0].Statement.Slot_index != proto.SCPSlotIndex(3) {
		t.Fatalf("first envelope slot mismatch: got %d, want 3", got[0].Statement.Slot_index)
	}
	if got[1].Statement.Slot_index != proto.SCPSlotIndex(4) || got[2].Statement.Slot_index != proto.SCPSlotIndex(4) {
		t.Fatalf("expected envelopes for slot 4 after first gap, got slots %d and %d", got[1].Statement.Slot_index, got[2].Statement.Slot_index)
	}

	afterGap := st.FindSCPEnvelopesSinceSlot(ctx, proto.SCPSlotIndex(6))
	if len(afterGap) != 1 || afterGap[0].Statement.Slot_index != proto.SCPSlotIndex(6) {
		t.Fatalf("expected single envelope at slot 6, got %+v", afterGap)
	}

	none := st.FindSCPEnvelopesSinceSlot(ctx, proto.SCPSlotIndex(10))
	if len(none) != 0 {
		t.Fatalf("expected no envelopes starting from slot 10, got %d", len(none))
	}
}

func TestFindConsensusValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		Freq:             time.Hour,
		MaxCatchupBlocks: 10,
		MaxSCPSlots:      10,
	}

	st := NewStorage(ctx, cfg)
	st.Start(ctx, dir)

	cv := newConsensusValue(time.Unix(1700001000, 0).UTC(), 10)
	st.PersistConsensusValue(ctx, cv)

	got := st.FindConsensusValue(ctx, cv.Nomination.Block_height)
	if got == nil {
		t.Fatalf("FindConsensusValue returned nil for persisted height %d", cv.Nomination.Block_height)
	}

	if !reflect.DeepEqual(*got, cv) {
		t.Fatalf("FindConsensusValue mismatch:\n got: %+v\nwant: %+v", *got, cv)
	}

	if missing := st.FindConsensusValue(ctx, cv.Nomination.Block_height+1); missing != nil {
		t.Fatalf("expected nil for non-existent height, got %+v", missing)
	}
}

func TestFindConsensusValuesAbove(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	cfg := Config{
		Freq:             time.Hour,
		MaxCatchupBlocks: 10,
		MaxSCPSlots:      10,
	}

	st := NewStorage(ctx, cfg)
	st.Start(ctx, dir)

	consensusValues := []proto.ConsensusValue{
		newConsensusValue(time.Unix(1700002000, 0).UTC(), 5),
		newConsensusValue(time.Unix(1700003000, 0).UTC(), 6),
		newConsensusValue(time.Unix(1700004000, 0).UTC(), 8),
	}
	for _, cv := range consensusValues {
		st.PersistConsensusValue(ctx, cv)
	}

	got := FindConsensusValuesAbove(ctx, st, 4)

	if len(got) != len(consensusValues) {
		t.Fatalf("expected %d consensus values above height 4, got %d", len(consensusValues), len(got))
	}

	for i, want := range consensusValues {
		if !reflect.DeepEqual(got[i], want) {
			t.Fatalf("consensus value mismatch at index %d:\n got: %+v\nwant: %+v", i, got[i], want)
		}
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
