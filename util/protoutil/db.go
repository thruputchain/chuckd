package protoutil

import (
	"fmt"

	proto "github.com/groundhogchain/protocol-go"
)

func DbTipString(tip proto.DbTip) string {
	return fmt.Sprintf("DbTip[id %d; height %d; value %s]",
		tip.Id, tip.Value.Proposal_header.Block_height, ValueDisplay(&tip.Value))
}
