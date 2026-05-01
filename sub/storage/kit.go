package storage

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

// XXX: interaction between storage and protocol upgrades?

type CatchupBlockPersistence interface {
	PrePersistCatchupBlock(
		ctx context.Context,
		cb proto.CatchupBlock,
		cvHash proto.ConsensusValueHash,
	)
	PersistAlreadyPrePersistedCatchupBlock(
		ctx context.Context,
		height proto.BlockHeight,
		cvHash proto.ConsensusValueHash,
	)
	LookupCatchupBlock(ctx context.Context, cvHash proto.ConsensusValueHash) *proto.CatchupBlock
	FindCatchupBlock(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock
}

type ConsensusValuePersistence interface {
	PersistCV(ctx context.Context, cv proto.ConsensusValue, hash proto.ConsensusValueHash)
	PrePersistCV(ctx context.Context, cv proto.ConsensusValue, hash proto.ConsensusValueHash)
	FindCV(ctx context.Context, height proto.BlockHeight, hash proto.ConsensusValueHash) *proto.ConsensusValue
	FindCommittedCV(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue
	FindHighestCommittedCV(ctx context.Context) *proto.ConsensusValue
}

func FindCommittedConsensusValuesAbove(ctx context.Context, x ConsensusValuePersistence, above proto.BlockHeight) []proto.ConsensusValue {
	found := []proto.ConsensusValue{}
	highest := x.FindHighestCommittedCV(ctx)
	if highest == nil {
		return nil
	}
	for height := above + 1; height <= highest.Nomination.Block_height; height++ {
		cv := x.FindCommittedCV(ctx, height)
		if cv == nil {
			continue
		}
		found = append(found, *cv)
	}
	return found
}

type SCPEnvelopePersistence interface {
	PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope)

	PersistSCPEnvelopeFromLocal(ctx context.Context, env proto.SCPEnvelope)

	FindSCPEnvelopesForSlot(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope

	FindSCPEnvelopesForSlotFromLocal(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope

	// ListSlotsWithSCPEnvelopesNotBelow lists all slots not below a given slot that have persisted scp envelopes.
	// The slots are returned in ascending order.
	ListSlotsWithSCPEnvelopesNotBelow(ctx context.Context, below proto.SCPSlotIndex) []proto.SCPSlotIndex
}

type Config struct {
	Freq             time.Duration // garbage collection loop freq
	MaxCatchupBlocks int           // catchup blocks
	MaxSCPSlots      int           // scp envelopes
}
