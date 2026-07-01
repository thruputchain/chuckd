package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/must"
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

	basis := newConsensusBasis(time.Unix(1700000000, 0).UTC(), 42)
	cb := proto.CatchupBlock{
		Consensus_basis:     basis,
		Signed_transactions: proto.ParallelSignedTransactionBatch{},
	}

	st.PersistCatchupBlock(ctx, cb, protoutil.ValueHashFromBasis(&basis))

	got, err := st.FindCatchupBlock(ctx, basis.Nomination.Block_height)
	if err != nil {
		t.Fatalf("FindCatchupBlock returned error: %v", err)
	}
	if got == nil {
		t.Fatalf("FindCatchupBlock returned no result")
	}

	if !reflect.DeepEqual(*got, cb) {
		t.Fatalf("FindCatchupBlock mismatch:\n got: %#v\nwant: %#v", *got, cb)
	}

	found, err := st.FindCatchupBlock(ctx, basis.Nomination.Block_height+1)
	if err != nil {
		t.Fatalf("FindCatchupBlock returned error: %v", err)
	}
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

	got, err := st.FindSCPEnvelopesForSlot(ctx, proto.SCPSlotIndex(4))
	if err != nil {
		t.Fatalf("FindSCPEnvelopesForSlot returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 envelopes, got %d", len(got))
	}

	if got[0].Statement.Slot_index != proto.SCPSlotIndex(4) {
		t.Fatalf("got %d, want 4", got[0].Statement.Slot_index)
	}
	if got[1].Statement.Slot_index != proto.SCPSlotIndex(4) {
		t.Fatalf("got %d, want 4", got[1].Statement.Slot_index)
	}

	afterGap, err := st.FindSCPEnvelopesForSlot(ctx, proto.SCPSlotIndex(6))
	if err != nil {
		t.Fatalf("FindSCPEnvelopesForSlot returned error: %v", err)
	}
	if len(afterGap) != 1 || afterGap[0].Statement.Slot_index != proto.SCPSlotIndex(6) {
		t.Fatalf("expected single envelope at slot 6, got %+v", afterGap)
	}

	none, err := st.FindSCPEnvelopesForSlot(ctx, proto.SCPSlotIndex(10))
	if err != nil {
		t.Fatalf("FindSCPEnvelopesForSlot returned error: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no envelopes starting from slot 10, got %d", len(none))
	}
}

func TestListSlotsWithSCPEnvelopesNotBelow(t *testing.T) {
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

	st.PersistSCPEnvelope(ctx, newEnvelope(3, 0x11))
	st.PersistSCPEnvelopeFromLocal(ctx, newEnvelope(4, 0x12))
	st.PersistSCPEnvelope(ctx, newEnvelope(6, 0x13))

	got, err := st.ListSlotsWithSCPEnvelopesNotBelow(ctx, 4)
	if err != nil {
		t.Fatalf("ListSlotsWithSCPEnvelopesNotBelow returned error: %v", err)
	}
	want := []proto.SCPSlotIndex{4, 6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListSlotsWithSCPEnvelopesNotBelow mismatch:\n got: %v\nwant: %v", got, want)
	}

	got, err = st.ListSlotsWithSCPEnvelopesNotBelow(ctx, 5)
	if err != nil {
		t.Fatalf("ListSlotsWithSCPEnvelopesNotBelow returned error: %v", err)
	}
	want = []proto.SCPSlotIndex{6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListSlotsWithSCPEnvelopesNotBelow mismatch:\n got: %v\nwant: %v", got, want)
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

	basis := newConsensusBasis(time.Unix(1700001000, 0).UTC(), 10)
	st.PersistCV(ctx, basis, protoutil.ValueHashFromBasis(&basis))

	got, err := st.FindCommittedCV(ctx, basis.Nomination.Block_height)
	if err != nil {
		t.Fatalf("FindCommittedCV returned error: %v", err)
	}
	if got == nil {
		t.Fatalf("FindConsensusValue returned nil for persisted height %d", basis.Nomination.Block_height)
	}

	if !reflect.DeepEqual(*got, basis) {
		t.Fatalf("FindConsensusValue mismatch:\n got: %+v\nwant: %+v", *got, basis)
	}

	missing, err := st.FindCommittedCV(ctx, basis.Nomination.Block_height+1)
	if err != nil {
		t.Fatalf("FindCommittedCV returned error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for non-existent height, got %+v", missing)
	}
}

func TestFindConsensusValuesAbove(t *testing.T) {
	// liblog.SetLogLevelDebug()
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

	consensusBases := []proto.ConsensusBasis{
		newConsensusBasis(time.Unix(1700002000, 0).UTC(), 5),
		newConsensusBasis(time.Unix(1700003000, 0).UTC(), 6),
		newConsensusBasis(time.Unix(1700004000, 0).UTC(), 8),
	}
	for _, basis := range consensusBases {
		st.PersistCV(ctx, basis, protoutil.ValueHashFromBasis(&basis))
	}

	got, err := FindCommittedConsensusValuesAbove(ctx, st, 4)
	if err != nil {
		t.Fatalf("FindCommittedConsensusValuesAbove returned error: %v", err)
	}

	if len(got) != len(consensusBases) {
		t.Fatalf("expected %d consensus values above height 4, got %d", len(consensusBases), len(got))
	}

	for i, want := range consensusBases {
		if !reflect.DeepEqual(got[i], want) {
			t.Fatalf("consensus bases mismatch at index %d:\n got: %+v\nwant: %+v", i, got[i], want)
		}
	}
}

func TestPersistAlreadyPrePersistedRejectsZeroByteFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	full := filepath.Join(dir, "value.xdr")
	prepersisted := prepersist(ctx, full)
	if err := os.WriteFile(prepersisted, nil, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	gotErr := must.Try(func() {
		NewPrePersist().PersistAlreadyPrePersisted(ctx, full)
	})
	if gotErr == nil || !strings.Contains(gotErr.Error(), "persisted storage file is empty") {
		t.Fatalf("expected zero-byte file error, got %v", gotErr)
	}
	if _, err := os.Stat(full); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no final file, got stat error %v", err)
	}
}

func TestFindCommittedCVRejectsZeroByteFile(t *testing.T) {
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
	defer st.Stop()

	basis := newConsensusBasis(time.Unix(1700006000, 0).UTC(), 12)
	full := st.ConsensusValueFilepath(basis.Nomination.Block_height, protoutil.ValueHashFromBasis(&basis))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(full, nil, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	gotErr := must.Try(func() {
		_, _ = st.FindCommittedCV(ctx, basis.Nomination.Block_height)
	})
	if gotErr == nil || !strings.Contains(gotErr.Error(), "persisted storage file is empty") {
		t.Fatalf("expected zero-byte file error, got %v", gotErr)
	}
}

func newConsensusBasis(ts time.Time, height int64) proto.ConsensusBasis {
	return proto.ConsensusBasis{
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
