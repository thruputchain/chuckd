package cvpool

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type CVPool interface {
	CVPoolAdd(
		ctx context.Context,
		basis proto.ConsensusBasis,
		value proto.ConsensusValue, // passed for efficiency purposes
	) error

	CVPoolCommit(
		ctx context.Context,
		basis proto.ConsensusBasis,
		value proto.ConsensusValue, // passed for efficiency purposes
	) error

	CVPoolLookup(
		ctx context.Context,
		value proto.ConsensusValue,
	) (*proto.ConsensusBasis, error)

	CVPoolWaitFor(
		ctx context.Context,
		value proto.ConsensusValue,
	) (*proto.ConsensusBasis, error)

	// height-dependent queries

	CVPoolFind(
		ctx context.Context,
		value proto.ConsensusValue,
	) (*proto.ConsensusBasis, error)

	CVPoolFindCommitted(
		ctx context.Context,
		height proto.BlockHeight,
	) (*proto.ConsensusBasis, error)

	CVPoolFindHighestCommitted(
		ctx context.Context,
	) (*proto.ConsensusBasis, error)

	CVPoolFindCommittedAbove(
		ctx context.Context,
		above proto.BlockHeight,
	) ([]proto.ConsensusBasis, error)
}
