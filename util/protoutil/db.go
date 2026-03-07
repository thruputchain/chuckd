package protoutil

import (
	"fmt"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func DbTipString(tip proto.DbTip) string {
	return fmt.Sprintf("DbTip[height %d; block_hash %s]",
		tip.Height, libxdr.HashString(tip.Block_hash))
}
