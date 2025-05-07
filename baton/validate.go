package baton

import proto "github.com/groundhogchain/protocol-go"

type ValidatableBlock struct {
	BlockHeight          proto.BlockHeight
	BlockHash            proto.BlockHash
	PrevBlockHash        proto.BlockHash
	ProposedTransactions []proto.ProposedTransaction
}
