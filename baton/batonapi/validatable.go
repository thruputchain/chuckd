package batonapi

import (
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: is this structure necessary any more?
type ValidatableBlock struct {
	FromCatchup   bool
	FromConsensus bool

	BlockHeight          proto.BlockHeight
	BlockHash            proto.BlockHash
	PrevBlockHash        proto.BlockHash
	ProposedTransactions []*proto.ProposedTransaction

	DbTip Tip

	// a copy of the latest nomination or recent validation, if it matches the block to validate;
	AlreadyValidatedNomination *proto.Nomination
	AlreadyValidatedBlockID    *proto.BlockID
}
