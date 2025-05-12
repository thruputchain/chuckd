package baton

import proto "github.com/groundhogchain/protocol-go"

type ValidatableBlock struct {
	FromCatchup   bool
	FromConsensus bool

	BlockHeight          proto.BlockHeight
	BlockHash            proto.BlockHash
	PrevBlockHash        proto.BlockHash
	ProposedTransactions []*proto.ProposedTransaction

	DbTipHeight proto.BlockHeight
	DbTipHash   proto.BlockHash
	DbTipID     proto.BlockID

	// a copy of the latest nomination or recent validation, if it matches the block to validate;
	AlreadyValidatedProposal *proto.Proposal
	AlreadyValidatedBlockID  *proto.BlockID
}
