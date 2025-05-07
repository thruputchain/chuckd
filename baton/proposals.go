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
	Slot     api.Slot
	Proposal *proto.Proposal
}

func (p proposal) BlockHeight() proto.BlockHeight {
	return p.Proposal.Block_height
}

func (p proposal) BlockHash() proto.BlockHash {
	return p.Proposal.Block_hash
}

func (p proposal) Proto() *proto.Proposal {
	return p.Proposal
}
