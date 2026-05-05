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
	)

	CVPoolCommit(
		ctx context.Context,
		cv proto.ConsensusValue,
		cue proto.ConsensusValueCue, // passed for efficiency purposes
	)

	CVPoolLookup(
		ctx context.Context,
		cue proto.ConsensusValueCue,
	) *proto.ConsensusValue

	CVPoolWaitFor(
		ctx context.Context,
		cue proto.ConsensusValueCue,
	) (*proto.ConsensusValue, error)

	// height-dependent queries

	CVPoolFind(
		ctx context.Context,
		cue proto.ConsensusValueCue,
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
