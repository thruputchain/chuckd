package baton

import (
	"fmt"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type externalized interface {
	Proposer() proto.ValidatorPublicInfo
	Height() proto.BlockHeight
	ConsensusValueHash() proto.ConsensusValueHash
	ConsensusValueCue() *proto.ConsensusValueCue
	PrevConsensusValueHash() proto.ConsensusValueHash
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
}

func externalizedStack(prev externalized, next externalized) bool {
	return next.Height() == prev.Height()+1 &&
		next.PrevBlockHash() == prev.BlockHash() &&
		next.PrevConsensusValueHash() == prev.ConsensusValueHash()
}

func externalizedAgree(ext1, ext2 externalized) bool {
	return ext1.Height() == ext2.Height() &&
		ext1.BlockHash() == ext2.BlockHash() &&
		ext1.PrevBlockHash() == ext2.PrevBlockHash() &&
		ext1.ConsensusValueHash() == ext2.ConsensusValueHash() &&
		ext1.PrevConsensusValueHash() == ext2.PrevConsensusValueHash()
}

func ExternalizedString(ext externalized) string {
	return fmt.Sprintf("Externalized[height %d; block_hash %s; prev_block_hash %s; cvh %s; prev_cvh %s]",
		ext.Height(), libxdr.HashString(ext.BlockHash()), libxdr.HashString(ext.PrevBlockHash()),
		libxdr.HashString(ext.ConsensusValueHash()), libxdr.HashString(ext.PrevConsensusValueHash()))
}

type externalizedConsensus struct {
	cue *proto.ConsensusValueCue
	cv  *proto.ConsensusValue
}

func newExternalized(
	cue *proto.ConsensusValueCue,
	cv *proto.ConsensusValue,
) externalizedConsensus {
	return externalizedConsensus{
		cue: cue,
		cv:  cv,
	}
}

func (x externalizedConsensus) Proposer() proto.ValidatorPublicInfo {
	return x.cv.Nomination.Proposer
}

func (x externalizedConsensus) ConsensusValue() *proto.ConsensusValue {
	return x.cv
}

func (x externalizedConsensus) ConsensusValueCue() *proto.ConsensusValueCue {
	return x.cue
}

func (x externalizedConsensus) ConsensusValueHash() proto.ConsensusValueHash {
	return x.cue.Cv_hash
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

func (x externalizedCue) ConsensusValueCue() *proto.ConsensusValueCue {
	return x.cvc
}

func (x externalizedCue) Proposer() proto.ValidatorPublicInfo {
	return x.cvc.Proposer
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
