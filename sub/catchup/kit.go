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
	)

	CatchupBlockPoolPersist(
		ctx context.Context,
		cv proto.CatchupBlock, //XXX?
	)

	CatchupBlockPoolLookup(
		ctx context.Context,
		height proto.BlockHeight,
	) *proto.CatchupBlock
}
