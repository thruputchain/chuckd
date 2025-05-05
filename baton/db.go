package baton

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) restoreDb(ctx context.Context) {
	var r dbRestored
	r.BlockHeight, r.BlockHash, r.Err = b.handler.RestoreDb(ctx)
	b.dbRestored <- r
}

type dbRestored struct {
	BlockHeight proto.BlockHeight
	BlockHash   proto.BlockHash
	Err         error
}

func (b *Baton) updateDbState(height proto.BlockHeight, hash proto.BlockHash) {
	b.db.lk.Lock()
	defer b.db.lk.Unlock()
	b.db.tipHeight = height
	b.db.tipHash = hash
}
