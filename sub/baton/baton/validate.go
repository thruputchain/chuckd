package baton

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) hashCV(cv *proto.ConsensusValue) proto.ConsensusValueHash {
	return protoutil.HashCV(cv)
}

func (b *Baton) cueFromConsensusValue(cv *proto.ConsensusValue) *proto.ConsensusValueCue {
	cvc := protoutil.CueFromConsensusValue(cv)
	return &cvc
}

// validated cache

func (b *Baton) setValidated(cv *proto.ConsensusValue, cvh proto.ConsensusValueHash, blockID proto.BlockID) {
	b.validated[cvh] = validated{
		ConsensusValue: cv,
		BlockID:        blockID,
	}
}

func (b *Baton) getValidatedConsensusFromCVH(cvh proto.ConsensusValueHash) *proto.ConsensusValue {
	v, ok := b.validated[cvh]
	if !ok {
		return nil
	}
	return v.ConsensusValue
}

func (b *Baton) getValidatedBlockIDFromCVH(cvh proto.ConsensusValueHash) *proto.BlockID {
	v, ok := b.validated[cvh]
	if !ok {
		return nil
	}
	return &v.BlockID
}

// invalidated cache

func (b *Baton) setInvalidated(cv *proto.ConsensusValue, cvh proto.ConsensusValueHash) {
	b.invalidated[cvh] = invalidated{ConsensusValue: cv}
}

func (b *Baton) isInvalidatedCVH(cvh proto.ConsensusValueHash) bool {
	_, ok := b.invalidated[cvh]
	return ok
}
