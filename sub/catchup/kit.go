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
	CatchupBlockPoolAdd(
		ctx context.Context,
		cv proto.CatchupBlock,
		cvHash proto.ConsensusValueHash,
	)

	CatchupBlockPoolPersist(
		ctx context.Context,
		height proto.BlockHeight,
		cvHash proto.ConsensusValueHash,
	)

	CatchupBlockPoolLookup(
		ctx context.Context,
		cvHash proto.ConsensusValueHash,
	) *proto.CatchupBlock

	CatchupBlockPoolFindHeight(
		ctx context.Context,
		height proto.BlockHeight,
	) *proto.CatchupBlock
}
