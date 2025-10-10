package kit

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type Submitter interface {
	SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction)
}
