package cvpool

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type CVPool interface {
	CVPoolAdd(
		ctx context.Context,
		cv proto.ConsensusValue,
		cvh proto.ConsensusValueHash,
	)

	CVPoolCommit(
		ctx context.Context,
		cv proto.ConsensusValue,
		cvh proto.ConsensusValueHash,
	)

	CVPoolLookup(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) *proto.ConsensusValue

	CVPoolWaitFor(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) (*proto.ConsensusValue, error)

	// height-dependent queries

	CVPoolFind(
		ctx context.Context,
		height proto.BlockHeight,
		hash proto.ConsensusValueHash,
	) *proto.ConsensusValue

	CVPoolFindCommitted(
		ctx context.Context,
		height proto.BlockHeight,
	) *proto.ConsensusValue

	CVPoolFindHighestCommitted(
		ctx context.Context,
	) *proto.ConsensusValue

	CVPoolFindCommittedAbove(
		ctx context.Context,
		above proto.BlockHeight,
	) []proto.ConsensusValue
}
