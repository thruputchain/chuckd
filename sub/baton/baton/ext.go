package baton

import (
	"fmt"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type externalized interface {
	Proposer() proto.ValidatorID
	Height() proto.BlockHeight
	ConsensusValue() *proto.ConsensusValue
	ValueHash() proto.ConsensusValueHash
	PrevValueHash() proto.ConsensusValueHash
}

func externalizedStack(prev externalized, next externalized) bool {
	return next.Height() == prev.Height()+1 &&
		next.PrevValueHash() == prev.ValueHash()
}

func externalizedAgree(ext1, ext2 externalized) bool {
	return ext1.Height() == ext2.Height() &&
		ext1.ValueHash() == ext2.ValueHash() &&
		ext1.PrevValueHash() == ext2.PrevValueHash()
}

func ExternalizedString(ext externalized) string {
	return fmt.Sprintf("Externalized[height %d; value_hash %s; prev_value_hash %s]",
		ext.Height(), libxdr.HashString(ext.ValueHash()), libxdr.HashString(ext.PrevValueHash()))
}

type externalizedBasis struct {
	value *proto.ConsensusValue
	basis *proto.ConsensusBasis
}

func newExternalized(
	value *proto.ConsensusValue,
	basis *proto.ConsensusBasis,
) externalizedBasis {
	return externalizedBasis{
		value: value,
		basis: basis,
	}
}

func (x externalizedBasis) Proposer() proto.ValidatorID {
	return x.value.Proposer
}

func (x externalizedBasis) ConsensusBasis() *proto.ConsensusBasis {
	return x.basis
}

func (x externalizedBasis) ConsensusValue() *proto.ConsensusValue {
	return x.value
}

func (x externalizedBasis) ValueHash() proto.ConsensusValueHash {
	return protoutil.HashValueToProto(x.value)
}

func (x externalizedBasis) PrevValueHash() proto.ConsensusValueHash {
	return x.value.Prev_value_hash
}

func (x externalizedBasis) Height() proto.BlockHeight {
	return x.value.Proposal_header.Block_height
}

type externalizedValue struct {
	height proto.BlockHeight
	value  *proto.ConsensusValue
}

func (x externalizedValue) ConsensusValue() *proto.ConsensusValue {
	return x.value
}

func (x externalizedValue) Proposer() proto.ValidatorID {
	return x.value.Proposer
}

func (x externalizedValue) ValueHash() proto.ConsensusValueHash {
	return protoutil.HashValueToProto(x.value)
}

func (x externalizedValue) PrevValueHash() proto.ConsensusValueHash {
	return x.value.Prev_value_hash
}

func (x externalizedValue) Height() proto.BlockHeight {
	return x.height
}
