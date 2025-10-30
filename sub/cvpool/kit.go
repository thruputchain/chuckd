package cvpool

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type CVPool interface {
	ConsensusValuePoolAdd(
		ctx context.Context,
		cv proto.ConsensusValue,
	)
	ConsensusValuePoolLookup(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) *proto.ConsensusValue
	ConsensusValuePoolWaitFor(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) (*proto.ConsensusValue, error)
}
