package baton

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) restoreDb(ctx context.Context) {
	var r dbRestored
	r.BlockHeight, r.BlockHash, r.BlockID, r.Err = b.handler.RestoreDb(ctx)
	if r.Err == nil {
		b.updateDbState(r.BlockHeight, r.BlockHash, r.BlockID)
	}
	b.dbRestored <- r
}

type dbRestored struct {
	BlockHeight proto.BlockHeight
	BlockHash   proto.BlockHash
	BlockID     proto.BlockID
	Err         error
}

func (b *Baton) updateDbState(height proto.BlockHeight, hash proto.BlockHash, id proto.BlockID) {
	b.db.lk.Lock()
	defer b.db.lk.Unlock()
	b.db.tipHeight = height
	b.db.tipHash = hash
	b.db.tipID = id
}
