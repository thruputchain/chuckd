package baton

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

type caughtUp struct{}

type catchupBlocks []stampedCatchupBlock

func (bs catchupBlocks) Empty() bool {
	return len(bs) == 0
}

func (bs catchupBlocks) BlockHeightBottom() proto.BlockHeight {
	return bs[0].BlockHeight()
}

func (bs catchupBlocks) BlockHeightTop() proto.BlockHeight {
	return bs[len(bs)-1].BlockHeight()
}

func (bs catchupBlocks) BlockHashBottom() proto.BlockHash {
	return bs[0].BlockHash()
}

func (bs catchupBlocks) BlockHashTop() proto.BlockHash {
	return bs[len(bs)-1].BlockHash()
}

type stampedCatchupBlock struct {
	stamp time.Time
	block *proto.CatchupBlock
}

func (b stampedCatchupBlock) BlockHeight() proto.BlockHeight {
	return b.block.Block_height
}

func (b stampedCatchupBlock) BlockHash() proto.BlockHash {
	return b.block.Block_hash
}

type catchupPending struct {
	onto proto.BlockHeight
}

// baton

func (b *Baton) refillCatchup(ctx context.Context) {

	b.lock()
	defer b.unlock()

	if b.scp.prop.Empty() {
		return // scp state not known yet
	}

	dbTip := b.db.tipHeight
	scpBottom := b.scp.prop.BlockHeightBottom()

	if dbTip+1 >= scpBottom {
		return // we are up to speed
	}

	XXX
}
