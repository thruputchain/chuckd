package cvpool

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type CVPool interface {
	ConsensusValuePoolAdd(
		ctx context.Context,
		cv proto.ConsensusValue,
		cvh proto.ConsensusValueHash,
	)

	ConsensusValuePoolPersist(
		ctx context.Context,
		cv proto.ConsensusValue,
		cvh proto.ConsensusValueHash,
	)

	ConsensusValuePoolLookupHash(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) *proto.ConsensusValue

	ConsensusValuePoolWaitFor(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) (*proto.ConsensusValue, error)

	// height-dependent queries

	ConsensusValuePoolFindHeight(
		ctx context.Context,
		height proto.BlockHeight,
	) *proto.ConsensusValue

	ConsensusValuePoolFindHighest(
		ctx context.Context,
	) *proto.ConsensusValue

	ConsensusValuePoolFindAbove(
		ctx context.Context,
		above proto.BlockHeight,
	) []proto.ConsensusValue
}
