package catchup

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/sub/storage"
	proto "github.com/groundhogchain/protocol-go"
)

type CatchupBlockPool struct {
	persist    storage.CatchupBlockPersistence
	lk         sync.Mutex
	persisting map[proto.BlockHash]chan struct{}
}

func NewCatchupBlockPool(ctx context.Context, persist storage.CatchupBlockPersistence) *CatchupBlockPool {
	cbp := &CatchupBlockPool{
		persist:    persist,
		persisting: map[proto.BlockHash]chan struct{}{},
	}
	var _ CBPool = cbp
	return cbp
}

func (cbp *CatchupBlockPool) CatchupBlockPoolAdd(
	ctx context.Context,
	cb proto.CatchupBlock,
) {
	cbp.lk.Lock()
	defer cbp.lk.Unlock()
	donePersisting := make(chan struct{})
	cbp.persisting[cb.Consensus_value.Nomination.Block_hash] = donePersisting

	go func() {
		cbp.persist.PrePersistCatchupBlock(context.WithoutCancel(ctx), cb)
		close(donePersisting)

		cbp.lk.Lock()
		defer cbp.lk.Unlock()
		delete(cbp.persisting, cb.Consensus_value.Nomination.Block_hash)
	}()
}

func (cbp *CatchupBlockPool) waitFinishPrePersist(_ context.Context, bh proto.BlockHash) {
	cbp.lk.Lock()
	donePersisting, ok := cbp.persisting[bh]
	cbp.lk.Unlock()
	if !ok {
		return
	}
	<-donePersisting
}

func (cbp *CatchupBlockPool) CatchupBlockPoolPersist(
	ctx context.Context,
	height proto.BlockHeight,
	hash proto.BlockHash,
) {
	cbp.waitFinishPrePersist(ctx, hash)
	cbp.persist.PersistAlreadyPrePersistedCatchupBlock(ctx, height, hash)
}

func (cbp *CatchupBlockPool) CatchupBlockPoolFindHeight(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.CatchupBlock {
	return cbp.persist.FindCatchupBlock(ctx, height)
}
