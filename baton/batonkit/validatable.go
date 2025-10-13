package batonkit

import (
	proto "github.com/groundhogchain/protocol-go"
)

type ValidatableBlock struct {
	Nomination       proto.Nomination
	Catchup          *proto.CatchupBlock
	AlreadyValidated *proto.BlockID
	Onto             proto.DbTip
}

func (vb *ValidatableBlock) BlockHeight() proto.BlockHeight {
	if vb.Catchup != nil {
		return vb.Catchup.Block_height
	} else {
		return vb.Nomination.Block_height
	}
}

func (vb *ValidatableBlock) BlockHash() proto.BlockHash {
	if vb.Catchup != nil {
		return vb.Catchup.Block_hash
	} else {
		return vb.Nomination.Block_hash
	}
}

func (vb *ValidatableBlock) PrevBlockHash() proto.BlockHash {
	if vb.Catchup != nil {
		return vb.Catchup.Prev_block_hash
	} else {
		return vb.Nomination.Prev_block_hash
	}
}
