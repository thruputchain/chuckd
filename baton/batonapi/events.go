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
	Height     proto.BlockHeight
	Hash       proto.BlockHash
}

func (x EventNominating) String() string {
	return fmt.Sprintf("NOMINATING [tx_count %d; height %d; hash %s]", x.NumTx, x.Height, form.Display(x.Hash))
}

type EventNominated struct {
	Nominated any
	Onto      Tip
	NumTx     int
	Height    proto.BlockHeight
	Hash      proto.BlockHash
}

func (x EventNominated) String() string {
	return fmt.Sprintf("NOMINATED [tx_count %d; height %d; hash %s]", x.NumTx, x.Height, form.Display(x.Hash))
}

type EventCommitted struct {
	Committed any
	Onto      Tip
	Height    proto.BlockHeight
	Hash      proto.BlockHash
}

func (x EventCommitted) String() string {
	return fmt.Sprintf("COMMITTED [height %d; hash %s]", x.Height, form.Display(x.Hash))
}

type EventValidating struct {
	Validating any
	Onto       Tip
	NumTx      int
	Height     proto.BlockHeight
	Hash       proto.BlockHash
}

func (x EventValidating) String() string {
	return fmt.Sprintf("VALIDATING [tx_count %d; height %d; hash %s]", x.NumTx, x.Height, form.Display(x.Hash))
}

type EventValidated struct {
	Validated any
	Onto      Tip
	NumTx     int
	Height    proto.BlockHeight
	Hash      proto.BlockHash
}

func (x EventValidated) String() string {
	return fmt.Sprintf("VALIDATED [tx_count %d; height %d; hash %s]", x.NumTx, x.Height, form.Display(x.Hash))
}
