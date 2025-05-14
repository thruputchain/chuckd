package baton

import (
	"context"
)

func (b *Baton) restoreDb(ctx context.Context) (Tip, error) {
	var tip Tip
	var err error
	tip.Height, tip.Hash, tip.ID, err = b.handler.RestoreDb(ctx)
	return tip, err
}
