package baton

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

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

// baton

func (b *Baton) refillCatchup(ctx context.Context) bool {

	b.lock()
	defer b.unlock()

	XXX
}
