package batonkit

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Committable interface {
	BlockHeight() proto.BlockHeight
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
	AlreadyValidated() *proto.BlockID
	AsNomination(m xdrutil.XdrMarshaler) *proto.Nomination
	SignedTransactions() []proto.SignedTransaction
}

type CommittableNomination struct {
	Nomination *proto.Nomination // always present
	Validated  *proto.BlockID    // always present
}

func (cb *CommittableNomination) BlockHeight() proto.BlockHeight {
	return cb.Nomination.Block_height
}

func (cb *CommittableNomination) BlockHash() proto.BlockHash {
	return cb.Nomination.Block_hash
}

func (cb *CommittableNomination) PrevBlockHash() proto.BlockHash {
	return cb.Nomination.Prev_block_hash
}

func (cb *CommittableNomination) AlreadyValidated() *proto.BlockID {
	if cb.Validated == nil {
		panic("bug")
	}
	return cb.Validated
}

func (cb *CommittableNomination) AsNomination(_ xdrutil.XdrMarshaler) *proto.Nomination {
	return cb.Nomination
}

func (cb *CommittableNomination) SignedTransactions() []proto.SignedTransaction {
	return nil
}

type CommittableCatchup struct {
	Catchup   *proto.CatchupBlock // always present
	Validated *proto.BlockID      // optional
}

func (cb *CommittableCatchup) BlockHeight() proto.BlockHeight {
	return cb.Catchup.Block_height
}

func (cb *CommittableCatchup) BlockHash() proto.BlockHash {
	return cb.Catchup.Block_hash
}

func (cb *CommittableCatchup) PrevBlockHash() proto.BlockHash {
	return cb.Catchup.Prev_block_hash
}

func (cb *CommittableCatchup) AlreadyValidated() *proto.BlockID {
	return cb.Validated
}

func (cb *CommittableCatchup) AsNomination(m xdrutil.XdrMarshaler) *proto.Nomination {
	return protoutil.NominationFromCatchup(m, cb.Catchup)
}

func (cb *CommittableCatchup) SignedTransactions() []proto.SignedTransaction {
	return cb.Catchup.Signed_transactions
}

type CommittableBlock struct {
	Committable
	Onto proto.DbTip
}
