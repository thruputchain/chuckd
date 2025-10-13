package protoutil

import (
	"fmt"

	"github.com/groundhogchain/chuckd/lib/form"
	proto "github.com/groundhogchain/protocol-go"
)

func DbTipString(tip proto.DbTip) string {
	return fmt.Sprintf("[height %d; hash %s]", tip.Height, form.Display(tip.Hash))
}
