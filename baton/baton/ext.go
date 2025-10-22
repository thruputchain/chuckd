package baton

import proto "github.com/groundhogchain/protocol-go"

type externalized struct {
	cv   *proto.ConsensusValue
	hash proto.ConsensusValueHash
}

func (b *Baton) newExternalized(cv *proto.ConsensusValue) *externalized {
	return &externalized{
		cv:   cv,
		hash: b.hashCV(cv),
	}
}

func (x *externalized) ConsensusValue() *proto.ConsensusValue {
	return x.cv
}

func (x *externalized) ConsensusValueHash() proto.ConsensusValueHash {
	return x.hash
}

func (x *externalized) Height() proto.BlockHeight {
	return x.cv.Nomination.Block_height
}

func (x *externalized) BlockHash() proto.BlockHash {
	return x.cv.Nomination.Block_hash
}

func (x *externalized) PrevBlockHash() proto.BlockHash {
	return x.cv.Nomination.Prev_block_hash
}
