package protoutil

import (
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

func proposalHashNumber(p *proto.Proposal) int64 {
	hash := p.Block_hash
	// convert first 8 bytes to int64
	var result int64
	for i := 0; i < 8; i++ {
		result |= int64(hash[i]) << (i * 8)
	}
	return result
}

type ProposalsByHash []*proto.Proposal

func (ps ProposalsByHash) Len() int {
	return len(ps)
}

func (ps ProposalsByHash) Less(i, j int) bool {
	return proposalHashNumber(ps[j]) < proposalHashNumber(ps[i])
}

func (ps ProposalsByHash) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func (ps ProposalsByHash) Sort() {
	sort.Sort(ps)
}
