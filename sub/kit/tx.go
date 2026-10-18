package kit

import (
	"context"

	proto "github.com/thruputchain/thruput-protocol-go"
)

type SignedTxSubmitter interface {
	SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction)
}
