package baton

import (
	"github.com/thruputchain/chuckd/util/protoutil"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// validated cache

type validated struct {
	*proto.ConsensusBasis
	*proto.ConsensusValue
	proto.BlockID
}

func (b *Baton) setValidated(
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
	blockID proto.BlockID,
) {
	b.validated[protoutil.HashValue(value)] = validated{
		ConsensusBasis: basis,
		ConsensusValue: value,
		BlockID:        blockID,
	}
}

func (b *Baton) getValidatedBasis(value *proto.ConsensusValue) *proto.ConsensusBasis {
	v, ok := b.validated[protoutil.HashValue(value)]
	if !ok {
		return nil
	}
	return v.ConsensusBasis
}

func (b *Baton) getValidatedBlockID(value *proto.ConsensusValue) *proto.BlockID {
	v, ok := b.validated[protoutil.HashValue(value)]
	if !ok {
		return nil
	}
	return &v.BlockID
}

// invalidated cache

type invalidated struct {
	*proto.ConsensusBasis
}

func (b *Baton) setInvalidated(basis *proto.ConsensusBasis, value *proto.ConsensusValue) {
	b.invalidated[protoutil.HashValue(value)] = invalidated{ConsensusBasis: basis}
}

func (b *Baton) isInvalidated(value *proto.ConsensusValue) bool {
	cv, ok := b.invalidated[protoutil.HashValue(value)]
	return ok && protoutil.ValueHeight(&cv.ConsensusBasis.Value) == protoutil.ValueHeight(value)
}
