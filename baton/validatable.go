package baton

import (
	"fmt"

	"github.com/groundhogchain/chuckd/lib/form"
	proto "github.com/groundhogchain/protocol-go"
)

type Tip struct {
	Height proto.BlockHeight
	Hash   proto.BlockHash
	ID     proto.BlockID
}

func (x Tip) String() string {
	return fmt.Sprintf("[%d; %s]", x.Height, form.SprintJSON(x.Hash))
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
