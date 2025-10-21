package storage

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

// XXX: interaction between storage and protocol upgrades?
// XXX: remove error return values
// XXX: %020d -> %012d

type CatchupBlockPersistence interface {
	PersistCatchupBlock(ctx context.Context, cb proto.CatchupBlock)
	FindCatchupBlock(ctx context.Context, height proto.BlockHeight) (proto.CatchupBlock, error)
}

type ConsensusValuePersistence interface {
	PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue)
	FindConsensusValue(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue
	FindConsensusValuesAbove(ctx context.Context, above proto.BlockHeight) []proto.ConsensusValue
}

type SCPEnvelopePersistence interface {
	PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope)
	FindSCPEnvelopesSinceSlot(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error)
}

type Config struct {
	Freq             time.Duration // garbage collection loop freq
	MaxCatchupBlocks int           // catchup blocks
	MaxSCPSlots      int           // scp envelopes
}
