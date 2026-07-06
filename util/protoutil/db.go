package protoutil

import (
	"fmt"

	proto "github.com/groundhogchain/protocol-go"
)

func TipHeight(tip proto.DbTip) proto.BlockHeight {
	return tip.Value.Proposal_header.Block_height
}

func DbTipString(tip proto.DbTip) string {
	return fmt.Sprintf("DbTip[height %d; value_hash %s]",
		TipHeight(tip), HashValue(&tip.Value).Display())
}
