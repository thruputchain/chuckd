package baton

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// validated cache

type validated struct {
	*proto.ConsensusBasis
	*proto.ConsensusValueCue
	proto.BlockID
}

func (b *Baton) setValidated(
	basis *proto.ConsensusBasis,
	cvc *proto.ConsensusValueCue,
	blockID proto.BlockID,
) {
	b.validated[protoutil.HashCVC(cvc)] = validated{
		ConsensusBasis:    basis,
		ConsensusValueCue: cvc,
		BlockID:           blockID,
	}
}

func (b *Baton) getValidatedBasis(cvc *proto.ConsensusValueCue) *proto.ConsensusBasis {
	v, ok := b.validated[protoutil.HashCVC(cvc)]
	if !ok {
		return nil
	}
	return v.ConsensusBasis
}

func (b *Baton) getValidatedBlockID(cvc *proto.ConsensusValueCue) *proto.BlockID {
	v, ok := b.validated[protoutil.HashCVC(cvc)]
	if !ok {
		return nil
	}
	return &v.BlockID
}

// invalidated cache

type invalidated struct {
	*proto.ConsensusBasis
}

func (b *Baton) setInvalidated(basis *proto.ConsensusBasis, cvc *proto.ConsensusValueCue) {
	b.invalidated[protoutil.HashCVC(cvc)] = invalidated{ConsensusBasis: basis}
}

func (b *Baton) isInvalidated(cvc *proto.ConsensusValueCue) bool {
	cv, ok := b.invalidated[protoutil.HashCVC(cvc)]
	return ok && cv.ConsensusBasis.Nomination.Block_height == cvc.Block_height
}
