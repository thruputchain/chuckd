package baton

import (
	"context"
)

type nominate struct{}

// XXX: mark committed transactions when committing to db
// XXX: abort nomination if consensus races ahead of us

func (b *Baton) startNominating(ctx context.Context) {
	for {

		// nominate signals that our db is up to speed;
		// there are no known externalizations above us
		<-b.nominate

		// pool all available transactions (our own + uncommitted from batches)
		txs := b.handler.ListUncommitedTxs(ctx)

		// prioritize and pick the top, applying any limits on transactions per block
		txs = b.handler.PrioritizeTxs(ctx, txs)

		// XXX
	}
}
