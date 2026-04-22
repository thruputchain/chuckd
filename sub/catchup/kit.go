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
	// CBPoolAdd initiates the pre-persisting of a block.
	CBPoolAdd(
		ctx context.Context,
		cv proto.CatchupBlock,
		cvHash proto.ConsensusValueHash,
	)

	// CBPoolWaitPrePersist ensures that a block completed pre-persisting.
	// CatchupBlockPoolAdd must have been called before this function can be called.
	CBPoolWaitPrePersist(
		ctx context.Context,
		height proto.BlockHeight,
		cvHash proto.ConsensusValueHash,
	)

	// CBPoolWaitPersist ensures that a block is persisted.
	// CatchupBlockPoolAdd must have been called before this function can be called.
	CBPoolWaitPersist(
		ctx context.Context,
		height proto.BlockHeight,
		cvHash proto.ConsensusValueHash,
	)

	CBPoolFindCommitted(
		ctx context.Context,
		height proto.BlockHeight,
	) *proto.CatchupBlock
}
