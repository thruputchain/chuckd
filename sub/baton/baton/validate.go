package baton

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) hashCV(cv *proto.ConsensusValue) proto.ConsensusValueHash {
	return protoutil.HashCV(b.cvM, cv)
}

func (b *Baton) cueFromConsensusValue(cv *proto.ConsensusValue) *proto.ConsensusValueCue {
	cvc := protoutil.CueFromConsensusValue(b.cvM, cv)
	return &cvc
}

// validated cache

func (b *Baton) setValidated(cv *proto.ConsensusValue, blockID proto.BlockID) {
	b.validated[b.hashCV(cv)] = validated{
		ConsensusValue: cv,
		BlockID:        blockID,
	}
}

func (b *Baton) getValidatedBlockIDFromCV(cv *proto.ConsensusValue) *proto.BlockID {
	return b.getValidatedBlockIDFromCVH(b.hashCV(cv))
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

func (b *Baton) setInvalidated(cv *proto.ConsensusValue) {
	b.invalidated[b.hashCV(cv)] = invalidated{ConsensusValue: cv}
}

func (b *Baton) isInvalidatedCV(cv *proto.ConsensusValue) bool {
	return b.isInvalidatedCVH(b.hashCV(cv))
}

func (b *Baton) isInvalidatedCVH(cvh proto.ConsensusValueHash) bool {
	_, ok := b.invalidated[cvh]
	return ok
}
