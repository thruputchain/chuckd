package batonkit

import (
	proto "github.com/groundhogchain/protocol-go"
)

type Committable interface {
	BlockHeight() proto.BlockHeight
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
	AlreadyValidated() *proto.BlockID
	AsConsensusValue() *proto.ConsensusValue
	SignedTransactions() []proto.SignedTransaction
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

func (cb *CommittableNomination) SignedTransactions() []proto.SignedTransaction {
	return nil
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

func (cb *CommittableCatchup) SignedTransactions() []proto.SignedTransaction {
	return cb.Catchup.Signed_transactions
}

type CommittableBlock struct {
	Committable
	Onto proto.DbTip
}
