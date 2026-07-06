package batonkit

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Committable interface {
	BlockHeight() proto.BlockHeight
	ValueHash() proto.ConsensusValueHash
	PrevValueHash() proto.ConsensusValueHash
	AlreadyValidated() *proto.BlockID
	AsConsensusBasis() *proto.ConsensusBasis
	ConsensusValue() *proto.ConsensusValue
	UnsubmittedSignedTransactions() []proto.SignedTransaction
	NumTransactions() int
}

type CommittableNomination struct {
	basis     *proto.ConsensusBasis // always present
	value     *proto.ConsensusValue
	validated *proto.BlockID        // always present
}

func NewCommittableNomination(
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
	validated *proto.BlockID,
) *CommittableNomination {
	return &CommittableNomination{
		basis:     basis,
		value:     value,
		validated: validated,
	}
}

func (cb *CommittableNomination) BlockHeight() proto.BlockHeight {
	return cb.value.Proposal_header.Block_height
}

func (cb *CommittableNomination) ValueHash() proto.ConsensusValueHash {
	return protoutil.HashValueToProto(cb.value)
}

func (cb *CommittableNomination) PrevValueHash() proto.ConsensusValueHash {
	return cb.value.Prev_value_hash
}

func (cb *CommittableNomination) AlreadyValidated() *proto.BlockID {
	if cb.validated == nil {
		panic("bug")
	}
	return cb.validated
}

func (cb *CommittableNomination) ConsensusValue() *proto.ConsensusValue {
	return cb.value
}

func (cb *CommittableNomination) AsConsensusBasis() *proto.ConsensusBasis {
	return cb.basis
}

func (cb *CommittableNomination) UnsubmittedSignedTransactions() []proto.SignedTransaction {
	return nil
}

func (cb *CommittableNomination) NumTransactions() int {
	return xdrutil.TokenSizeOfBatch(cb.basis.Proposal_txs.Hash)
}

type CommittableCatchup struct {
	catchup   *proto.CatchupBlock // always present
	value     *proto.ConsensusValue
	txs       []proto.SignedTransaction
	validated *proto.BlockID // optional
}

func NewCommittableCatchup(
	catchup *proto.CatchupBlock,
	value *proto.ConsensusValue,
	txs []proto.SignedTransaction,
	validated *proto.BlockID,
) *CommittableCatchup {
	return &CommittableCatchup{
		catchup:   catchup,
		value:     value,
		txs:       txs,
		validated: validated,
	}
}

func (cb *CommittableCatchup) BlockHeight() proto.BlockHeight {
	return cb.value.Proposal_header.Block_height
}

func (cb *CommittableCatchup) ValueHash() proto.ConsensusValueHash {
	return protoutil.HashValueToProto(cb.value)
}

func (cb *CommittableCatchup) PrevValueHash() proto.ConsensusValueHash {
	return cb.value.Prev_value_hash
}

func (cb *CommittableCatchup) AlreadyValidated() *proto.BlockID {
	return cb.validated
}

func (cb *CommittableCatchup) ConsensusValue() *proto.ConsensusValue {
	return cb.value
}

func (cb *CommittableCatchup) AsConsensusBasis() *proto.ConsensusBasis {
	return &cb.catchup.Consensus_basis
}

func (cb *CommittableCatchup) UnsubmittedSignedTransactions() []proto.SignedTransaction {
	return cb.txs
}

func (cb *CommittableCatchup) NumTransactions() int {
	return len(cb.txs)
}

type CommittableBlock struct {
	Committable
	Onto proto.DbTip
}
