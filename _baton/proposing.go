package _baton

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type proposals []*proposal

func (ps proposals) Empty() bool {
	return len(ps) == 0
}

func (ps proposals) BlockHeightBottom() proto.BlockHeight {
	return ps[0].BlockHeight()
}

type proposal struct {
	ext extProp
}

func (p *proposal) BlockHeight() proto.BlockHeight {
	return p.ext.Proposal.Block_height
}

func (p *proposal) BlockHash() proto.BlockHash {
	return p.ext.Proposal.Block_hash
}

func (b *Baton) startProposing(ctx context.Context) {
	XXX
}
