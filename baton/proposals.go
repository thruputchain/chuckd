package baton

import (
	proto "github.com/groundhogchain/protocol-go"
)

type proposals []*proposal

func (ps proposals) IsEmpty() bool {
	return len(ps) == 0
}

type proposal struct {
	ext extProp
}

func (p proposal) BlockHeight() proto.BlockHeight {
	return p.ext.Proposal.Block_height
}

func (p proposal) BlockHash() proto.BlockHash {
	return p.ext.Proposal.Block_hash
}

func (p proposal) Proto() *proto.Proposal {
	return p.ext.Proposal
}
