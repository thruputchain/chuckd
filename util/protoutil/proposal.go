package protoutil

import (
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

func ProposalSize(p *proto.Proposal) int {
	s := 0
	for _, b := range p.Proposed_transaction_batches {
		s += int(b.Batch_size)
	}
	return s
}

type ProposalsBySize []*proto.Proposal

func (ps ProposalsBySize) Len() int {
	return len(ps)
}

func (ps ProposalsBySize) Less(i, j int) bool {
	return ProposalSize(ps[j]) < ProposalSize(ps[i])
}

func (ps ProposalsBySize) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func (ps ProposalsBySize) Sort() {
	sort.Sort(ps)
}
