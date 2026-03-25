package catchup

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type CatchupAgent interface {
	GetCatchupSources(context.Context, int64) []proto.ValidatorPublicInfo
	DialCatchup(context.Context, proto.ValidatorPublicInfo) (CatchupConn, error)
}

type CBPool interface {
	// CatchupBlockPoolAdd initiates the pre-persisting of a block.
	CatchupBlockPoolAdd(
		ctx context.Context,
		cv proto.CatchupBlock,
		cvHash proto.ConsensusValueHash,
	)

	// CatchupBlockPoolPrePersist ensures that a block completed pre-persisting.
	CatchupBlockPoolPrePersist(
		ctx context.Context,
		height proto.BlockHeight,
		cvHash proto.ConsensusValueHash,
	)

	// CatchupBlockPoolPersist ensures that a block is persisted.
	CatchupBlockPoolPersist(
		ctx context.Context,
		height proto.BlockHeight,
		cvHash proto.ConsensusValueHash,
	)

	// CatchupBlockPoolLookup will discover any block that was previously persisted or pre-persisted.
	CatchupBlockPoolLookup(
		ctx context.Context,
		cvHash proto.ConsensusValueHash,
	) *proto.CatchupBlock

	CatchupBlockPoolFindHeight(
		ctx context.Context,
		height proto.BlockHeight,
	) *proto.CatchupBlock
}
