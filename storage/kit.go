package storage

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

// TODO: garbage collect catchup blocks, scp envelopes

type Storage interface {
	StorageReader
	StorageWriter
}

type StorageWriter interface {
	AddCatchupBlock(ctx context.Context, cb proto.CatchupBlock)
	AddConsensusValue(ctx context.Context, cv proto.ConsensusValue)
	AddSCPEnvelope(ctx context.Context, stamp time.Time, seqno int64, env proto.SCPEnvelope)
}

type StorageReader interface {
	FindCatchupBlock(ctx context.Context, height proto.BlockHeight) (proto.CatchupBlock, error)
	LatestConsensusValues(ctx context.Context, n int) ([]proto.ConsensusValue, error)
	SCPEnvelopesSince(ctx context.Context, since time.Time) ([]proto.SCPEnvelope, error)
}
