package protoutil

import (
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func NominationFromCatchup(m xdrutil.XdrMarshaler, c *proto.CatchupBlock) *proto.Nomination {
	return &proto.Nomination{
		Proposer:                c.Proposer,
		Block_height:            c.Block_height,
		Block_hash:              c.Block_hash,
		Prev_block_hash:         c.Prev_block_hash,
		Transaction_hashes:      HashSignedTxs(m, c.Signed_transactions),
		Transaction_nominations: c.Nominated_transactions,
	}
}
