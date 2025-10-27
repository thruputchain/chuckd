package cvpool

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type CVPool interface {
	LookupConsensusValue(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) *proto.ConsensusValue
	WaitForConsensusValue(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) (*proto.ConsensusValue, error)
}
