package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/must"
)

func (b *Baton) restoreDb(ctx context.Context, restored chan<- synchronize) {
	var tip Tip
	var err error
	tip.Height, tip.Hash, tip.ID, err = b.handler.RestoreDb(ctx)
	restored <- func() {
		must.Assertf(ctx, err == nil, "db not restored: %v", err)
		b.tip = tip
	}
}
