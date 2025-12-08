package protoutil

import (
	"fmt"

	"github.com/groundhogchain/chuckd/lib/libtext"
	proto "github.com/groundhogchain/protocol-go"
)

func DbTipString(tip proto.DbTip) string {
	return fmt.Sprintf("[height %d; hash %s]", tip.Height, libtext.Display(tip.Hash))
}
