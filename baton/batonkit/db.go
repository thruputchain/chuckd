package batonkit

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

func FromDbTip(tip proto.DbTip) Tip {
	return Tip{Height: tip.Height, Hash: tip.Hash, ID: tip.Id}
}

func (x Tip) Proto() proto.DbTip {
	return proto.DbTip{Height: x.Height, Hash: x.Hash, Id: x.ID}
}

func (x Tip) String() string {
	return fmt.Sprintf("Tip{height:%d; hash:%s}", x.Height, form.Display(x.Hash))
}
