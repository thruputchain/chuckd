package cvpool

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type CVPool interface {
	ConsensusValuePoolAddWithHash(
		ctx context.Context,
		cv proto.ConsensusValue,
		cvh proto.ConsensusValueHash,
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
