package batonapi

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
	return fmt.Sprintf("Tip{height:%d; hash:%s}", x.Height, form.Display(x.Hash))
}
