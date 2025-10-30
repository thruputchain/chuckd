package baton

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// VTL (Validation Tail Length) is the number of slots below and including the last externalized slot that
// the node supports validation requests for.
const VTL = 3

func (b *Baton) hashCV(cv *proto.ConsensusValue) proto.ConsensusValueHash {
	return protoutil.HashCV(b.cvM, cv)
}

// validated cache

func (b *Baton) setValidated(cv *proto.ConsensusValue, blockID proto.BlockID) {
	b.validated[b.hashCV(cv)] = validated{
		ConsensusValue: cv,
		BlockID:        blockID,
	}
}

func (b *Baton) getValidatedCV(cv *proto.ConsensusValue) *proto.BlockID {
	return b.getValidatedCVH(b.hashCV(cv))
}

func (b *Baton) getValidatedCVH(cvh proto.ConsensusValueHash) *proto.BlockID {
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
