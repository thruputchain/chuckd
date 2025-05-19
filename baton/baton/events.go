package baton

import (
	"fmt"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/form"
	proto "github.com/groundhogchain/protocol-go"
)

type EventNominating struct {
	Nominating any
	Tip        sdk.Tip
	NumTx      int
	Proposal   *proto.Proposal
}

func (x EventNominating) String() string {
	return fmt.Sprintf("NOMINATING [%d; %s]", x.Proposal.Block_height, form.SprintJSON(x.Proposal.Block_hash))
}

type EventNominated struct {
	Nominated any
	Tip       sdk.Tip
	NumTx     int
	Proposal  *proto.Proposal
}

func (x EventNominated) String() string {
	return fmt.Sprintf("NOMINATED [%d; %s]", x.Proposal.Block_height, form.SprintJSON(x.Proposal.Block_hash))
}

type EventCommitted struct {
	Committed any
	*sdk.ValidatableBlock
	proto.BlockID
}

func (x EventCommitted) String() string {
	return fmt.Sprintf("COMMITTED [%d; %s]", x.ValidatableBlock.BlockHeight, form.SprintJSON(x.ValidatableBlock.BlockHash))
}

type EventValidating struct {
	Validating any
	*sdk.ValidatableBlock
	proto.BlockID
}

func (x EventValidating) String() string {
	return fmt.Sprintf("VALIDATING [%d; %s]", x.ValidatableBlock.BlockHeight, form.SprintJSON(x.ValidatableBlock.BlockHash))
}

type EventValidated struct {
	Validated any
	*sdk.ValidatableBlock
	proto.BlockID
}

func (x EventValidated) String() string {
	return fmt.Sprintf("VALIDATED [%d; %s]", x.ValidatableBlock.BlockHeight, form.SprintJSON(x.ValidatableBlock.BlockHash))
}
