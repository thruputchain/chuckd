package kit

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type SignedTxSubmitter interface {
	SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction)
}
