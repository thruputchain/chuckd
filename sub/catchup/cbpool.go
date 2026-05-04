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
	persisting map[protoutil.CVCHash]chan struct{}
}

func NewCatchupBlockPool(ctx context.Context, persist storage.CatchupBlockPersistence) *CatchupBlockPool {
	cbp := &CatchupBlockPool{
		persist:    persist,
		persisting: map[protoutil.CVCHash]chan struct{}{},
	}
	var _ CBPool = cbp
	return cbp
}

func (cbp *CatchupBlockPool) CBPoolAdd(
	ctx context.Context,
	cb proto.CatchupBlock,
	cue proto.ConsensusValueCue,
) {
	cbp.lk.Lock()
	defer cbp.lk.Unlock()

	liblog.Infof(ctx, "adding catchup block to pool: height %d; cv_hash %s; proposer %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cue.Cv_hash),
		protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))

	if _, alreadyPersisting := cbp.persisting[cue.Cv_hash]; alreadyPersisting {
		liblog.Infof(ctx, "catchup block already persisting: height %d; cv_hash %s; proposer %s",
			cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cue.Cv_hash),
			protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))
		return
	}

	donePersisting := make(chan struct{})
	cvch := protoutil.HashCVC(&cue)
	cbp.persisting[cvch] = donePersisting

	go func() {
		cbp.persist.PrePersistCatchupBlock(context.WithoutCancel(ctx), cb, cvch)
		close(donePersisting)

		liblog.Debugf(ctx, "finished pre-persisting catchup block: height %d; cv_hash %s; proposer %s",
			cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cue.Cv_hash),
			protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))

		cbp.lk.Lock()
		defer cbp.lk.Unlock()
		delete(cbp.persisting, cvch)
	}()
}

func (cbp *CatchupBlockPool) waitFinishPrePersist(ctx context.Context, cue proto.ConsensusValueCue) {
	cvch := protoutil.HashCVC(&cue)

	cbp.lk.Lock()
	donePersisting, ok := cbp.persisting[cvch]
	cbp.lk.Unlock()
	if !ok {
		liblog.Infof(ctx, "catchup need not wait for pre-persist: cv_hash %s",
			libxdr.HashString(cue.Cv_hash))
		return
	}
	<-donePersisting

	liblog.Infof(ctx, "finished waiting for catchup to pre-persist: cv_hash %s",
		libxdr.HashString(cue.Cv_hash))
}

func (cbp *CatchupBlockPool) CBPoolWaitPrePersist(
	ctx context.Context,
	cue proto.ConsensusValueCue,
) {
	liblog.Infof(ctx, "waiting for pre-persisting of catchup block: height %d; cv_hash %s",
		cue.Block_height, libxdr.HashString(cue.Cv_hash))

	cbp.waitFinishPrePersist(ctx, cue)
	// NOTE: we have no way of knowing currently if the block had initiated pre-persist at all.
}

func (cbp *CatchupBlockPool) CBPoolWaitPersist(
	ctx context.Context,
	cue proto.ConsensusValueCue,
) {
	liblog.Infof(ctx, "waiting for persisting of catchup block: height %d; cv_hash %s",
		cue.Block_height, libxdr.HashString(cue.Cv_hash))

	cbp.waitFinishPrePersist(ctx, cue)
	cbp.persist.PersistAlreadyPrePersistedCatchupBlock(ctx, cue.Block_height, protoutil.HashCVC(&cue))
}

// CBPoolLookup finds persisted or pre-persisted catchup blocks with given cv hash.
// Blocks we have nominated but are not committed will be found in pre-persist.
func (cbp *CatchupBlockPool) CBPoolLookup(
	ctx context.Context,
	cue proto.ConsensusValueCue,
) *proto.CatchupBlock {
	cbp.waitFinishPrePersist(ctx, cue)
	cb := cbp.persist.LookupCatchupBlock(ctx, protoutil.HashCVC(&cue))
	if cb == nil {
		return nil
	}
	if cb.Consensus_value.Nomination.Block_height != cue.Block_height {
		return nil
	}
	return cb
}

func (cbp *CatchupBlockPool) CBPoolFindCommitted(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.CatchupBlock {
	return cbp.persist.FindCatchupBlock(ctx, height)
}
