package cvpool

import (
	"context"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type CVPool interface {
	ConsensusValuePoolAdd(
		ctx context.Context,
		cv proto.ConsensusValue,
		cvh proto.ConsensusValueHash,
	)
	ConsensusValuePoolLookup(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) CVBox
	ConsensusValuePoolWaitFor(
		ctx context.Context,
		hash proto.ConsensusValueHash,
	) (CVBox, error)
}

type CVBox interface {
	CV() *proto.ConsensusValue
	NS() protoutil.NominationSerialized
}
