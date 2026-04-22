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
	persisting map[proto.ConsensusValueHash]chan struct{}
}

func NewCatchupBlockPool(ctx context.Context, persist storage.CatchupBlockPersistence) *CatchupBlockPool {
	cbp := &CatchupBlockPool{
		persist:    persist,
		persisting: map[proto.ConsensusValueHash]chan struct{}{},
	}
	var _ CBPool = cbp
	return cbp
}

func (cbp *CatchupBlockPool) CBPoolAdd(
	ctx context.Context,
	cb proto.CatchupBlock,
	cvHash proto.ConsensusValueHash,
) {
	cbp.lk.Lock()
	defer cbp.lk.Unlock()

	liblog.Infof(ctx, "adding catchup block to pool: height %d; cv_hash %s; proposer %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash),
		protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))

	if _, alreadyPersisting := cbp.persisting[cvHash]; alreadyPersisting {
		liblog.Infof(ctx, "catchup block already persisting: height %d; cv_hash %s; proposer %s",
			cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash),
			protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))
		return
	}

	donePersisting := make(chan struct{})
	cbp.persisting[cvHash] = donePersisting

	go func() {
		cbp.persist.PrePersistCatchupBlock(context.WithoutCancel(ctx), cb, cvHash)
		close(donePersisting)

		liblog.Debugf(ctx, "finished pre-persisting catchup block: height %d; cv_hash %s; proposer %s",
			cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash),
			protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))

		cbp.lk.Lock()
		defer cbp.lk.Unlock()
		delete(cbp.persisting, cvHash)
	}()
}

func (cbp *CatchupBlockPool) waitFinishPrePersist(ctx context.Context, cvHash proto.ConsensusValueHash) {
	cbp.lk.Lock()
	donePersisting, ok := cbp.persisting[cvHash]
	cbp.lk.Unlock()
	if !ok {
		liblog.Infof(ctx, "catchup need not wait for pre-persist: cv_hash %s",
			libxdr.HashString(cvHash))
		return
	}
	<-donePersisting

	liblog.Infof(ctx, "finished waiting for catchup to pre-persist: cv_hash %s",
		libxdr.HashString(cvHash))
}

func (cbp *CatchupBlockPool) CBPoolWaitPrePersist(
	ctx context.Context,
	height proto.BlockHeight,
	cvHash proto.ConsensusValueHash,
) {
	liblog.Infof(ctx, "waiting for pre-persisting of catchup block: height %d; cv_hash %s",
		height, libxdr.HashString(cvHash))

	cbp.waitFinishPrePersist(ctx, cvHash)
	// NOTE: we have no way of knowing currently if the block had initiated pre-persist at all.
}

func (cbp *CatchupBlockPool) CBPoolWaitPersist(
	ctx context.Context,
	height proto.BlockHeight,
	cvHash proto.ConsensusValueHash,
) {
	liblog.Infof(ctx, "waiting for persisting of catchup block: height %d; cv_hash %s",
		height, libxdr.HashString(cvHash))

	cbp.waitFinishPrePersist(ctx, cvHash)
	cbp.persist.PersistAlreadyPrePersistedCatchupBlock(ctx, height, cvHash)
}

func (cbp *CatchupBlockPool) CBPoolFindCommitted(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.CatchupBlock {
	return cbp.persist.FindCatchupBlock(ctx, height)
}
