package baton

import (
	"context"
)

func (b *Baton) restoreDb(ctx context.Context) {
	var r dbRestored
	r.Tip.Height, r.Tip.Hash, r.Tip.ID, r.Err = b.handler.RestoreDb(ctx)
	if r.Err == nil {
		b.updateDbState(r.Tip)
	}
	b.dbRestored <- r
}

type dbRestored struct {
	Tip Tip
	Err error
}

func (b *Baton) updateDbState(tip Tip) {
	b.db.lk.Lock()
	defer b.db.lk.Unlock()
	b.db.tip = tip
}
