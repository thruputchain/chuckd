package storage

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

type StorageWriter interface {
	AddCatchupBlock(ctx context.Context, cb proto.CatchupBlock)
	AddConsensusValue(ctx context.Context, cv proto.ConsensusValue)
	AddSCPEnvelope(ctx context.Context, stamp time.Time, seqno int64, env proto.SCPEnvelope)
}

type StorageReader interface {
	FindCatchupBlock(ctx context.Context, height proto.BlockHeight) (proto.CatchupBlock, error)
	LatestConsensusValues(ctx context.Context, intoReverseChrono []proto.ConsensusValue) error
	SCPEnvelopesSince(ctx context.Context, since time.Time) ([]proto.SCPEnvelope, error)
}
