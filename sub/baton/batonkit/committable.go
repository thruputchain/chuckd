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
	AsConsensusValue() *proto.ConsensusValue
	UnsubmittedSignedTransactions() []proto.SignedTransaction
	SignedTransactionHashes(m libxdr.XdrMarshaler) []proto.SignedTransactionHash
}

type CommittableNomination struct {
	ConsensusValue *proto.ConsensusValue // always present
	Validated      *proto.BlockID        // always present
}

func (cb *CommittableNomination) BlockHeight() proto.BlockHeight {
	return cb.ConsensusValue.Nomination.Block_height
}

func (cb *CommittableNomination) BlockHash() proto.BlockHash {
	return cb.ConsensusValue.Nomination.Block_hash
}

func (cb *CommittableNomination) PrevBlockHash() proto.BlockHash {
	return cb.ConsensusValue.Nomination.Prev_block_hash
}

func (cb *CommittableNomination) AlreadyValidated() *proto.BlockID {
	if cb.Validated == nil {
		panic("bug")
	}
	return cb.Validated
}

func (cb *CommittableNomination) AsConsensusValue() *proto.ConsensusValue {
	return cb.ConsensusValue
}

func (cb *CommittableNomination) UnsubmittedSignedTransactions() []proto.SignedTransaction {
	return nil
}

func (cb *CommittableNomination) SignedTransactionHashes(m libxdr.XdrMarshaler) []proto.SignedTransactionHash {
	return cb.ConsensusValue.Nomination.Transaction_hashes
}

type CommittableCatchup struct {
	Catchup   *proto.CatchupBlock // always present
	Validated *proto.BlockID      // optional
}

func (cb *CommittableCatchup) BlockHeight() proto.BlockHeight {
	return cb.Catchup.Consensus_value.Nomination.Block_height
}

func (cb *CommittableCatchup) BlockHash() proto.BlockHash {
	return cb.Catchup.Consensus_value.Nomination.Block_hash
}

func (cb *CommittableCatchup) PrevBlockHash() proto.BlockHash {
	return cb.Catchup.Consensus_value.Nomination.Prev_block_hash
}

func (cb *CommittableCatchup) AlreadyValidated() *proto.BlockID {
	return cb.Validated
}

func (cb *CommittableCatchup) AsConsensusValue() *proto.ConsensusValue {
	return &cb.Catchup.Consensus_value
}

func (cb *CommittableCatchup) UnsubmittedSignedTransactions() []proto.SignedTransaction {
	return cb.Catchup.Signed_transactions
}

func (cb *CommittableCatchup) SignedTransactionHashes(m libxdr.XdrMarshaler) []proto.SignedTransactionHash {
	return protoutil.HashSignedTxs(m, cb.Catchup.Signed_transactions)
}

type CommittableBlock struct {
	Committable
	Onto proto.DbTip
}
