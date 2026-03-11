package catchup

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/storage"
	proto "github.com/groundhogchain/protocol-go"
)

type CatchupBlockPool struct {
	persist storage.CatchupBlockPersistence
}

func NewCatchupBlockPool(ctx context.Context, persist storage.CatchupBlockPersistence) *CatchupBlockPool {
	cbp := &CatchupBlockPool{
		persist: persist,
	}
	var _ CBPool = cbp
	return cbp
}

func (cbp *CatchupBlockPool) CatchupBlockPoolAdd(
	ctx context.Context,
	cb proto.CatchupBlock,
) {
	go cbp.persist.PrePersistCatchupBlock(context.WithoutCancel(ctx), cb)
}

func (cbp *CatchupBlockPool) CatchupBlockPoolPersist(
	ctx context.Context,
	cb proto.CatchupBlock,
) {
	cbp.persist.PersistCatchupBlock(ctx, cb)
}

func (cbp *CatchupBlockPool) CatchupBlockPoolLookup(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.CatchupBlock {
	return cbp.persist.FindCatchupBlock(ctx, height)
}
