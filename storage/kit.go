package storage

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

// XXX: interaction between storage and protocol upgrades?

type CatchupBlockPersistence interface {
	PersistCatchupBlock(ctx context.Context, cb proto.CatchupBlock) // index: slot
	FindCatchupBlock(ctx context.Context, height proto.BlockHeight) (proto.CatchupBlock, error)
}

type ConsensusValuePersistence interface {
	PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue) // index: slot
	FindLatestConsensusValues(ctx context.Context, n int) ([]proto.ConsensusValue, error)
}

type SCPEnvelopePersistence interface {
	PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) // index: slot, time, hash
	FindSCPEnvelopesSinceSlot(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error)
}

type Config struct {
	Freq             time.Duration // garbage collection loop freq
	MaxCatchupBlocks int           // catchup blocks
	MaxSCPSlots      int           // scp envelopes
}
