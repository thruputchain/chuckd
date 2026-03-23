package catchup

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
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

	liblog.Infof("adding catchup block to pool: height %d; block_hash %s; proposer %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cb.Consensus_value.Nomination.Block_hash),
		// libxdr.HashString(protoutil.HashCVFast(&cb.Consensus_value)),
		protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))

	donePersisting := make(chan struct{})
	cbp.persisting[cb.Consensus_value.Nomination.Block_hash] = donePersisting

	go func() {
		cbp.persist.PrePersistCatchupBlock(context.WithoutCancel(ctx), cb)
		close(donePersisting)

		liblog.Infof("finished pre-persisting catchup block: height %d; block_hash %s; proposer %s",
			cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cb.Consensus_value.Nomination.Block_hash),
			// libxdr.HashString(protoutil.HashCVFast(&cb.Consensus_value)),
			protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))

		cbp.lk.Lock()
		defer cbp.lk.Unlock()
		delete(cbp.persisting, cb.Consensus_value.Nomination.Block_hash)
	}()
}

func (cbp *CatchupBlockPool) waitFinishPrePersist(_ context.Context, hash proto.BlockHash) {
	cbp.lk.Lock()
	donePersisting, ok := cbp.persisting[hash]
	cbp.lk.Unlock()
	if !ok {
		liblog.Infof("catchup need not wait for pre-persist: block_hash %s",
			libxdr.HashString(hash))
		return
	}
	<-donePersisting

	liblog.Infof("finished waiting for catchup to pre-persist: block_hash %s",
		libxdr.HashString(hash))
}

func (cbp *CatchupBlockPool) CatchupBlockPoolPersist(
	ctx context.Context,
	height proto.BlockHeight,
	hash proto.BlockHash,
) {
	liblog.Infof("persisting catchup block: height %d; block_hash %s",
		height, libxdr.HashString(hash))

	cbp.waitFinishPrePersist(ctx, hash)
	cbp.persist.PersistAlreadyPrePersistedCatchupBlock(ctx, height, hash)
}

func (cbp *CatchupBlockPool) CatchupBlockPoolFindHeight(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.CatchupBlock {
	return cbp.persist.FindCatchupBlock(ctx, height)
}
