package baton

import (
	"github.com/groundhogchain/chuckd/scp/api"
	proto "github.com/groundhogchain/protocol-go"
)

type proposals []proposal

func (ps proposals) IsEmpty() bool {
	return len(ps) == 0
}

type proposal struct {
	slot  api.Slot
	proto *proto.Proposal
}

func (p proposal) BlockHeight() proto.BlockHeight {
	return p.proto.Block_height
}

func (p proposal) BlockHash() proto.BlockHash {
	return p.proto.Block_hash
}

func (p proposal) PrevBlockHash() proto.BlockHash {
	return p.proto.Prev_block_hash
}

func (p proposal) Proto() *proto.Proposal {
	return p.proto
}
