package baton

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) hashCV(cv *proto.ConsensusValue) proto.ConsensusValueHash {
	return protoutil.HashCVFast(cv)
}

// validated cache

func (b *Baton) setValidated(
	cv *proto.ConsensusValue,
	cvh proto.ConsensusValueHash,
	height proto.BlockHeight,
	blockID proto.BlockID,
) {
	b.validated[cvh] = validated{
		ConsensusValue: cv,
		BlockHeight:    height,
		BlockID:        blockID,
	}
}

func (b *Baton) getValidatedCV(cvh proto.ConsensusValueHash, height proto.BlockHeight) *proto.ConsensusValue {
	v, ok := b.validated[cvh]
	if !ok || v.BlockHeight != height {
		return nil
	}
	return v.ConsensusValue
}

func (b *Baton) getValidatedBlockID(cvh proto.ConsensusValueHash, height proto.BlockHeight) *proto.BlockID {
	v, ok := b.validated[cvh]
	if !ok || v.BlockHeight != height {
		return nil
	}
	return &v.BlockID
}

// invalidated cache

func (b *Baton) setInvalidated(cv *proto.ConsensusValue, cvh proto.ConsensusValueHash) {
	b.invalidated[cvh] = invalidated{ConsensusValue: cv}
}

func (b *Baton) isInvalidated(cvh proto.ConsensusValueHash, height proto.BlockHeight) bool {
	cv, ok := b.invalidated[cvh]
	return ok && cv.ConsensusValue.Nomination.Block_height == height
}
