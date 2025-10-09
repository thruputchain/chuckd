package batonapi

import (
	"fmt"

	"github.com/groundhogchain/chuckd/lib/form"
	proto "github.com/groundhogchain/protocol-go"
)

type EventNominating struct {
	Nominating any
	Onto       Tip
	NumTx      int
	Nomination *proto.Nomination
}

func (x EventNominating) String() string {
	return fmt.Sprintf("NOMINATING [%d; %s]", x.Nomination.Block_height, form.Display(x.Nomination.Block_hash))
}

type EventNominated struct {
	Nominated  any
	Onto       Tip
	NumTx      int
	Nomination *proto.Nomination
}

func (x EventNominated) String() string {
	return fmt.Sprintf("NOMINATED [%d; %s]", x.Nomination.Block_height, form.Display(x.Nomination.Block_hash))
}

type EventCommitted struct {
	Committed any
	*ValidatableBlock
	proto.BlockID
}

func (x EventCommitted) String() string {
	return fmt.Sprintf("COMMITTED [%d; %s]", x.ValidatableBlock.BlockHeight, form.Display(x.ValidatableBlock.BlockHash))
}

type EventValidating struct {
	Validating any
	*ValidatableBlock
	proto.BlockID
}

func (x EventValidating) String() string {
	return fmt.Sprintf("VALIDATING [%d; %s]", x.ValidatableBlock.BlockHeight, form.Display(x.ValidatableBlock.BlockHash))
}

type EventValidated struct {
	Validated any
	*ValidatableBlock
	proto.BlockID
}

func (x EventValidated) String() string {
	return fmt.Sprintf("VALIDATED [%d; %s]", x.ValidatableBlock.BlockHeight, form.Display(x.ValidatableBlock.BlockHash))
}
