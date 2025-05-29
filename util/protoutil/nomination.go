package protoutil

import (
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

func nominationHashNumber(p *proto.Nomination) int64 {
	hash := p.Block_hash
	// convert first 8 bytes to int64
	var result int64
	for i := 0; i < 8; i++ {
		result |= int64(hash[i]) << (i * 8)
	}
	return result
}

type NominationsByHash []*proto.Nomination

func (ps NominationsByHash) Len() int {
	return len(ps)
}

func (ps NominationsByHash) Less(i, j int) bool {
	return nominationHashNumber(ps[j]) < nominationHashNumber(ps[i])
}

func (ps NominationsByHash) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func (ps NominationsByHash) Sort() {
	sort.Sort(ps)
}
