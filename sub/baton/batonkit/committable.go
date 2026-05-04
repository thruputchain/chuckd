package batonkit

import (
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Committable interface {
	BlockHeight() proto.BlockHeight
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
	AlreadyValidated() *proto.BlockID
	ConsensusValueHash() proto.ConsensusValueHash
	AsConsensusValue() *proto.ConsensusValue
	ConsensusValueCue() proto.ConsensusValueCue
	UnsubmittedSignedTransactions() []proto.SignedTransaction
	NumTransactions() int
}

type CommittableNomination struct {
	cv        *proto.ConsensusValue // always present
	cvc       proto.ConsensusValueCue
	validated *proto.BlockID // always present
}

func NewCommittableNomination(
	cv *proto.ConsensusValue,
	cvc proto.ConsensusValueCue,
	validated *proto.BlockID,
) *CommittableNomination {
	return &CommittableNomination{
		cv:        cv,
		cvc:       cvc,
		validated: validated,
	}
}

func (cb *CommittableNomination) BlockHeight() proto.BlockHeight {
	return cb.cv.Nomination.Block_height
}

func (cb *CommittableNomination) BlockHash() proto.BlockHash {
	return cb.cv.Nomination.Block_hash
}

func (cb *CommittableNomination) PrevBlockHash() proto.BlockHash {
	return cb.cv.Nomination.Prev_block_hash
}

func (cb *CommittableNomination) AlreadyValidated() *proto.BlockID {
	if cb.validated == nil {
		panic("bug")
	}
	return cb.validated
}

func (cb *CommittableNomination) ConsensusValueHash() proto.ConsensusValueHash {
	return cb.cvc.Cv_hash
}

func (cb *CommittableNomination) ConsensusValueCue() proto.ConsensusValueCue {
	return cb.cvc
}

func (cb *CommittableNomination) AsConsensusValue() *proto.ConsensusValue {
	return cb.cv
}

func (cb *CommittableNomination) UnsubmittedSignedTransactions() []proto.SignedTransaction {
	return nil
}

func (cb *CommittableNomination) NumTransactions() int {
	return xdrutil.TokenSizeOfBatch(cb.cv.Nomination.Transactions_hash)
}

type CommittableCatchup struct {
	catchup   *proto.CatchupBlock // always present
	cvc       proto.ConsensusValueCue
	txs       []proto.SignedTransaction
	validated *proto.BlockID // optional
}

func NewCommittableCatchup(
	catchup *proto.CatchupBlock,
	cvc proto.ConsensusValueCue,
	txs []proto.SignedTransaction,
	validated *proto.BlockID,
) *CommittableCatchup {
	return &CommittableCatchup{
		catchup:   catchup,
		cvc:       cvc,
		txs:       txs,
		validated: validated,
	}
}

func (cb *CommittableCatchup) BlockHeight() proto.BlockHeight {
	return cb.catchup.Consensus_value.Nomination.Block_height
}

func (cb *CommittableCatchup) BlockHash() proto.BlockHash {
	return cb.catchup.Consensus_value.Nomination.Block_hash
}

func (cb *CommittableCatchup) PrevBlockHash() proto.BlockHash {
	return cb.catchup.Consensus_value.Nomination.Prev_block_hash
}

func (cb *CommittableCatchup) AlreadyValidated() *proto.BlockID {
	return cb.validated
}

func (cb *CommittableCatchup) ConsensusValueHash() proto.ConsensusValueHash {
	return cb.cvc.Cv_hash
}

func (cb *CommittableCatchup) ConsensusValueCue() proto.ConsensusValueCue {
	return cb.cvc
}

func (cb *CommittableCatchup) AsConsensusValue() *proto.ConsensusValue {
	return &cb.catchup.Consensus_value
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
