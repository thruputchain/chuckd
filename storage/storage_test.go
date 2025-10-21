package storage

import (
	"context"
	"errors"
	"os"
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
