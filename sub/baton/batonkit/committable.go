package batonkit

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type Committable interface {
	BlockHeight() proto.BlockHeight
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
	AlreadyValidated() *proto.BlockID
	ConsensusValueHash() proto.ConsensusValueHash
	AsConsensusValue() *proto.ConsensusValue
	UnsubmittedSignedTransactions() []proto.SignedTransaction
	SignedTransactionHashes(m libxdr.XdrMarshaler) []proto.SignedTransactionHash
	NumTransactions() int
}

type CommittableNomination struct {
	cv        *proto.ConsensusValue // always present
	cvhash    proto.ConsensusValueHash
	validated *proto.BlockID // always present
}

func NewCommittableNomination(
	cv *proto.ConsensusValue,
	cvh proto.ConsensusValueHash,
	validated *proto.BlockID,
) *CommittableNomination {
	return &CommittableNomination{cv: cv, cvhash: cvh, validated: validated}
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
	return cb.cvhash
}

func (cb *CommittableNomination) AsConsensusValue() *proto.ConsensusValue {
	return cb.cv
}

func (cb *CommittableNomination) UnsubmittedSignedTransactions() []proto.SignedTransaction {
	return nil
}

func (cb *CommittableNomination) SignedTransactionHashes(m libxdr.XdrMarshaler) []proto.SignedTransactionHash {
	return cb.cv.Nomination.Transaction_hashes
}

func (cb *CommittableNomination) NumTransactions() int {
	return len(cb.cv.Nomination.Transaction_hashes)
}

type CommittableCatchup struct {
	catchup   *proto.CatchupBlock // always present
	cvhash    proto.ConsensusValueHash
	validated *proto.BlockID // optional
}

func NewCommittableCatchup(
	catchup *proto.CatchupBlock,
	cvhash proto.ConsensusValueHash,
	validated *proto.BlockID,
) *CommittableCatchup {
	return &CommittableCatchup{
		catchup:   catchup,
		cvhash:    cvhash,
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
	return cb.cvhash
}

func (cb *CommittableCatchup) AsConsensusValue() *proto.ConsensusValue {
	return &cb.catchup.Consensus_value
}

func (cb *CommittableCatchup) UnsubmittedSignedTransactions() []proto.SignedTransaction {
	return cb.catchup.Signed_transactions
}

func (cb *CommittableCatchup) SignedTransactionHashes(m libxdr.XdrMarshaler) []proto.SignedTransactionHash {
	return protoutil.HashSignedTxs(m, cb.catchup.Signed_transactions)
}

func (cb *CommittableCatchup) NumTransactions() int {
	return len(cb.catchup.Signed_transactions)
}

type CommittableBlock struct {
	Committable
	Onto proto.DbTip
}
