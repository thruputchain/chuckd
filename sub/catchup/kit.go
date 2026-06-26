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
		cb proto.CatchupBlock,
		value proto.ConsensusValue,
	) error

	// CBPoolWaitPrePersist ensures that a block completed pre-persisting.
	// CatchupBlockPoolAdd must have been called before this function can be called.
	CBPoolWaitPrePersist(
		ctx context.Context,
		value proto.ConsensusValue,
	) error

	// CBPoolWaitPersist ensures that a block is persisted.
	// CatchupBlockPoolAdd must have been called before this function can be called.
	CBPoolWaitPersist(
		ctx context.Context,
		value proto.ConsensusValue,
	) error

	// CBPoolLookup will discover any block that was previously persisted or pre-persisted.
	CBPoolLookup(
		ctx context.Context,
		value proto.ConsensusValue,
	) (*proto.CatchupBlock, error)

	CBPoolFindCommitted(
		ctx context.Context,
		height proto.BlockHeight,
	) (*proto.CatchupBlock, error)
}
