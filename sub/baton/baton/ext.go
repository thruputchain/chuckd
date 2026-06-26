package baton

import (
	"fmt"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type externalized interface {
	Proposer() proto.ValidatorPublicInfo
	Height() proto.BlockHeight
	ConsensusBasisHash() proto.ConsensusBasisHash
	ConsensusValueCue() *proto.ConsensusValueCue
	PrevConsensusBasisHash() proto.ConsensusBasisHash
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
}

func externalizedStack(prev externalized, next externalized) bool {
	return next.Height() == prev.Height()+1 &&
		next.PrevBlockHash() == prev.BlockHash() &&
		next.PrevConsensusBasisHash() == prev.ConsensusBasisHash()
}

func externalizedAgree(ext1, ext2 externalized) bool {
	return ext1.Height() == ext2.Height() &&
		ext1.BlockHash() == ext2.BlockHash() &&
		ext1.PrevBlockHash() == ext2.PrevBlockHash() &&
		ext1.ConsensusBasisHash() == ext2.ConsensusBasisHash() &&
		ext1.PrevConsensusBasisHash() == ext2.PrevConsensusBasisHash()
}

func ExternalizedString(ext externalized) string {
	return fmt.Sprintf("Externalized[height %d; block_hash %s; prev_block_hash %s; cvh %s; prev_cvh %s]",
		ext.Height(), libxdr.HashString(ext.BlockHash()), libxdr.HashString(ext.PrevBlockHash()),
		libxdr.HashString(ext.ConsensusBasisHash()), libxdr.HashString(ext.PrevConsensusBasisHash()))
}

type externalizedConsensus struct {
	cue   *proto.ConsensusValueCue
	basis *proto.ConsensusBasis
}

func newExternalized(
	cue *proto.ConsensusValueCue,
	cv *proto.ConsensusBasis,
) externalizedConsensus {
	return externalizedConsensus{
		cue:   cue,
		basis: cv,
	}
}

func (x externalizedConsensus) Proposer() proto.ValidatorPublicInfo {
	return x.basis.Nomination.Proposer
}

func (x externalizedConsensus) ConsensusBasis() *proto.ConsensusBasis {
	return x.basis
}

func (x externalizedConsensus) ConsensusValueCue() *proto.ConsensusValueCue {
	return x.cue
}

func (x externalizedConsensus) ConsensusBasisHash() proto.ConsensusBasisHash {
	return x.cue.Cv_hash
}

func (x externalizedConsensus) PrevConsensusBasisHash() proto.ConsensusBasisHash {
	return x.basis.Prev_cv_hash
}

func (x externalizedConsensus) Height() proto.BlockHeight {
	return x.basis.Nomination.Block_height
}

func (x externalizedConsensus) BlockHash() proto.BlockHash {
	return x.basis.Nomination.Block_hash
}

func (x externalizedConsensus) PrevBlockHash() proto.BlockHash {
	return x.basis.Nomination.Prev_block_hash
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

func (x externalizedCue) ConsensusBasisHash() proto.ConsensusBasisHash {
	return x.cvc.Cv_hash
}

func (x externalizedCue) PrevConsensusBasisHash() proto.ConsensusBasisHash {
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
