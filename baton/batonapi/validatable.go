package batonapi

import (
	proto "github.com/groundhogchain/protocol-go"
)

type ValidatableBlock struct {
	FromCatchup   bool
	FromConsensus bool

	BlockHeight          proto.BlockHeight
	BlockHash            proto.BlockHash
	PrevBlockHash        proto.BlockHash
	ProposedTransactions []*proto.ProposedTransaction

	DbTip Tip

	AlreadyValidatedBlockID *proto.BlockID
}
