package storage

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

// XXX: interaction between storage and protocol upgrades?

type CatchupBlockPersistence interface {
	PrePersistCatchupBlock(ctx context.Context, cb proto.CatchupBlock)
	PersistAlreadyPrePersistedCatchupBlock(ctx context.Context, height proto.BlockHeight, hash proto.BlockHash)
	FindCatchupBlock(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock
}

type ConsensusValuePersistence interface {
	PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue)
	PrePersistConsensusValue(ctx context.Context, cv proto.ConsensusValue)
	FindConsensusValue(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue
	FindHighestConsensusValue(ctx context.Context) *proto.ConsensusValue
}

func FindConsensusValuesAbove(ctx context.Context, x ConsensusValuePersistence, above proto.BlockHeight) []proto.ConsensusValue {
	found := []proto.ConsensusValue{}
	highest := x.FindHighestConsensusValue(ctx)
	if highest == nil {
		return nil
	}
	for height := above + 1; height <= highest.Nomination.Block_height; height++ {
		cv := x.FindConsensusValue(ctx, height)
		if cv == nil {
			continue
		}
		found = append(found, *cv)
	}
	return found
}

type SCPEnvelopePersistence interface {
	PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope)
	FindSCPEnvelopesSinceSlot(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope
}

type Config struct {
	Freq             time.Duration // garbage collection loop freq
	MaxCatchupBlocks int           // catchup blocks
	MaxSCPSlots      int           // scp envelopes
}
