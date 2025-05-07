package baton

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/xid"
)

type caughtUp struct{}

type catchupBlocks []catchupBlock

func (bs catchupBlocks) IsEmpty() bool {
	return len(bs) == 0
}

func (bs catchupBlocks) Find(h proto.BlockHeight) (catchupBlock, bool) {
	for _, cb := range bs {
		if cb.BlockHeight() == h {
			return cb, true
		}
	}
	return catchupBlock{}, false
}

type catchupBlock struct {
	source proto.ValidatorID
	stamp  time.Time
	block  *proto.CatchupBlock
}

func (b catchupBlock) Source() proto.ValidatorID {
	return b.source
}

func (b catchupBlock) BlockHeight() proto.BlockHeight {
	return b.block.Block_height
}

func (b catchupBlock) BlockHash() proto.BlockHash {
	return b.block.Block_hash
}

func (b catchupBlock) PrevBlockHash() proto.BlockHash {
	return b.block.Prev_block_hash
}

type catchupRequest struct {
	validator proto.ValidatorID
	from      proto.BlockHeight
	to        proto.BlockHeight
}

// baton

func (b *Baton) findCatchupBlocksAtHeight(h proto.BlockHeight) (catchupBlock, bool) {
	for _, cbs := range b.catchup.blocks {
		if cb, ok := cbs.Find(h); ok {
			return cb, true
		}
	}
	return catchupBlock{}, false
}

func (b *Baton) findCatchupRequestsAtHeight(h proto.BlockHeight) []xid.ID {
	var reqs []xid.ID
	for id, q := range b.catchup.pending {
		if q.from <= h && h < q.to {
			reqs = append(reqs, id)
		}
	}
	return reqs
}

func (b *Baton) requestCatchup(ctx context.Context) {

	b.lock()
	defer b.unlock()

	panic("XXX")
}
