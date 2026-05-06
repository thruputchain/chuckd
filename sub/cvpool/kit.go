package cvpool

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type CVPool interface {
	CVPoolAdd(
		ctx context.Context,
		cv proto.ConsensusValue,
		cue proto.ConsensusValueCue, // passed for efficiency purposes
	) error

	CVPoolCommit(
		ctx context.Context,
		cv proto.ConsensusValue,
		cue proto.ConsensusValueCue, // passed for efficiency purposes
	) error

	CVPoolLookup(
		ctx context.Context,
		cue proto.ConsensusValueCue,
	) (*proto.ConsensusValue, error)

	CVPoolWaitFor(
		ctx context.Context,
		cue proto.ConsensusValueCue,
	) (*proto.ConsensusValue, error)

	// height-dependent queries

	CVPoolFind(
		ctx context.Context,
		cue proto.ConsensusValueCue,
	) (*proto.ConsensusValue, error)

	CVPoolFindCommitted(
		ctx context.Context,
		height proto.BlockHeight,
	) (*proto.ConsensusValue, error)

	CVPoolFindHighestCommitted(
		ctx context.Context,
	) (*proto.ConsensusValue, error)

	CVPoolFindCommittedAbove(
		ctx context.Context,
		above proto.BlockHeight,
	) ([]proto.ConsensusValue, error)
}
