package baton

import proto "github.com/groundhogchain/protocol-go"

type externalized interface {
	ConsensusValueHash() proto.ConsensusValueHash
	PrevConsensusValueHash() proto.ConsensusValueHash
	Height() proto.BlockHeight
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
}

type externalizedConsensus struct {
	cv   *proto.ConsensusValue
	hash proto.ConsensusValueHash
}

func (b *Baton) newExternalized(cv *proto.ConsensusValue, cvh proto.ConsensusValueHash) externalizedConsensus {
	return externalizedConsensus{
		cv:   cv,
		hash: cvh,
	}
}

func (x externalizedConsensus) ConsensusValue() *proto.ConsensusValue {
	return x.cv
}

func (x externalizedConsensus) ConsensusValueHash() proto.ConsensusValueHash {
	return x.hash
}

func (x externalizedConsensus) PrevConsensusValueHash() proto.ConsensusValueHash {
	return x.cv.Prev_cv_hash
}

func (x externalizedConsensus) Height() proto.BlockHeight {
	return x.cv.Nomination.Block_height
}

func (x externalizedConsensus) BlockHash() proto.BlockHash {
	return x.cv.Nomination.Block_hash
}

func (x externalizedConsensus) PrevBlockHash() proto.BlockHash {
	return x.cv.Nomination.Prev_block_hash
}

type externalizedCue struct {
	height proto.BlockHeight
	cvc    *proto.ConsensusValueCue
}

func (x externalizedCue) ConsensusValueHash() proto.ConsensusValueHash {
	return x.cvc.Cv_hash
}

func (x externalizedCue) PrevConsensusValueHash() proto.ConsensusValueHash {
	return x.cvc.Prev_cv_hash
}

func (x externalizedCue) Height() proto.BlockHeight {
	return x.height
}

func (x externalizedCue) BlockHash() proto.BlockHash {
	return x.cvc.Block_hash
}

func (x externalizedCue) PrevBlockHash() proto.BlockHash {
	return x.cvc.Prev_block_hash
}
