package baton

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// validated cache

func (b *Baton) setValidated(
	cv *proto.ConsensusValue,
	cvc *proto.ConsensusValueCue,
	blockID proto.BlockID,
) {
	b.validated[protoutil.HashCVC(cvc)] = validated{
		ConsensusValue: cv,
		BlockHeight:    cvc.Block_height,
		BlockID:        blockID,
	}
}

func (b *Baton) getValidatedCV(cvc *proto.ConsensusValueCue) *proto.ConsensusValue {
	v, ok := b.validated[protoutil.HashCVC(cvc)]
	if !ok || v.BlockHeight != cvc.Block_height {
		return nil
	}
	return v.ConsensusValue
}

func (b *Baton) getValidatedBlockID(cvc *proto.ConsensusValueCue) *proto.BlockID {
	v, ok := b.validated[protoutil.HashCVC(cvc)]
	if !ok || v.BlockHeight != cvc.Block_height {
		return nil
	}
	return &v.BlockID
}

// invalidated cache

func (b *Baton) setInvalidated(cv *proto.ConsensusValue, cvc *proto.ConsensusValueCue) {
	b.invalidated[protoutil.HashCVC(cvc)] = invalidated{ConsensusValue: cv}
}

func (b *Baton) isInvalidated(cvc *proto.ConsensusValueCue) bool {
	cv, ok := b.invalidated[protoutil.HashCVC(cvc)]
	return ok && cv.ConsensusValue.Nomination.Block_height == cvc.Block_height
}
