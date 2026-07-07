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

type CommittableProposal struct {
	basis     *proto.ConsensusBasis // always present
	value     *proto.ConsensusValue
	validated *proto.BlockID // always present
}

func NewCommittableProposal(
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
	validated *proto.BlockID,
) *CommittableProposal {
	return &CommittableProposal{
		basis:     basis,
		value:     value,
		validated: validated,
	}
}

func (cb *CommittableProposal) BlockHeight() proto.BlockHeight {
	return cb.basis.Value.Proposal_header.Block_height
}

func (cb *CommittableProposal) ValueHash() proto.ConsensusValueHash {
	return protoutil.HashValueToProto(cb.value)
}

func (cb *CommittableProposal) PrevValueHash() proto.ConsensusValueHash {
	return cb.value.Prev_hash
}

func (cb *CommittableProposal) AlreadyValidated() *proto.BlockID {
	if cb.validated == nil {
		panic("bug")
	}
	return cb.validated
}

func (cb *CommittableProposal) ConsensusValue() *proto.ConsensusValue {
	return cb.value
}

func (cb *CommittableProposal) AsConsensusBasis() *proto.ConsensusBasis {
	return cb.basis
}

func (cb *CommittableProposal) UnsubmittedSignedTransactions() []proto.SignedTransaction {
	return nil
}

func (cb *CommittableProposal) NumTransactions() int {
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
	return cb.catchup.Consensus_basis.Value.Proposal_header.Block_height
}

func (cb *CommittableCatchup) ValueHash() proto.ConsensusValueHash {
	return protoutil.HashValueToProto(cb.value)
}

func (cb *CommittableCatchup) PrevValueHash() proto.ConsensusValueHash {
	return cb.value.Prev_hash
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
