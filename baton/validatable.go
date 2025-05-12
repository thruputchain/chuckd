package baton

import proto "github.com/groundhogchain/protocol-go"

type Tip struct {
	Height proto.BlockHeight
	Hash   proto.BlockHash
	ID     proto.BlockID
}

type ValidatableBlock struct {
	FromCatchup   bool
	FromConsensus bool

	BlockHeight          proto.BlockHeight
	BlockHash            proto.BlockHash
	PrevBlockHash        proto.BlockHash
	ProposedTransactions []*proto.ProposedTransaction

	DbTip Tip

	// a copy of the latest nomination or recent validation, if it matches the block to validate;
	AlreadyValidatedProposal *proto.Proposal
	AlreadyValidatedBlockID  *proto.BlockID
}
