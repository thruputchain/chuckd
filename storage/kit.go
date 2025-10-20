package storage

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

// XXX: interaction between storage and protocol upgrades?

type Storage interface {
	StorageReader
	StorageWriter
}

type StorageWriter interface {
	AddCatchupBlock(ctx context.Context, cb proto.CatchupBlock)     // index: slot
	AddConsensusValue(ctx context.Context, cv proto.ConsensusValue) // index: slot
	AddSCPEnvelope(ctx context.Context, env proto.SCPEnvelope)      // index: slot, time, hash
}

type StorageReader interface {
	FindCatchupBlock(ctx context.Context, height proto.BlockHeight) (proto.CatchupBlock, error)
	FindLatestConsensusValues(ctx context.Context, n int) ([]proto.ConsensusValue, error)
	FindSCPEnvelopesSinceSlot(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error)
}

type Config struct {
	Freq             time.Duration // garbage collection loop freq
	MaxCatchupBlocks int           // catchup blocks
	MaxSCPSlots      int           // scp envelopes
}
