package batonkit

import (
	proto "github.com/groundhogchain/protocol-go"
)

type CommittableBlock struct {
	Nomination       proto.Nomination
	Catchup          *proto.CatchupBlock
	AlreadyValidated *proto.BlockID
	Onto             proto.DbTip
}

func (cb *CommittableBlock) BlockHeight() proto.BlockHeight {
	return cb.Nomination.Block_height
}

func (cb *CommittableBlock) BlockHash() proto.BlockHash {
	return cb.Nomination.Block_hash
}

func (cb *CommittableBlock) PrevBlockHash() proto.BlockHash {
	return cb.Nomination.Prev_block_hash
}
