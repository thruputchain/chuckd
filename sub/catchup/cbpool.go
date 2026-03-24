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

func (cbp *CatchupBlockPool) CatchupBlockPoolAdd(
	ctx context.Context,
	cb proto.CatchupBlock,
	cvHash proto.ConsensusValueHash,
) {
	cbp.lk.Lock()
	defer cbp.lk.Unlock()

	liblog.Infof("adding catchup block to pool: height %d; cv_hash %s; proposer %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash),
		protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))

	donePersisting := make(chan struct{})
	cbp.persisting[cvHash] = donePersisting

	go func() {
		cbp.persist.PrePersistCatchupBlock(context.WithoutCancel(ctx), cb, cvHash)
		close(donePersisting)

		liblog.Infof("finished pre-persisting catchup block: height %d; cv_hash %s; proposer %s",
			cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash),
			protoutil.VPublicShortString(cb.Consensus_value.Nomination.Proposer))

		cbp.lk.Lock()
		defer cbp.lk.Unlock()
		delete(cbp.persisting, cvHash)
	}()
}

func (cbp *CatchupBlockPool) waitFinishPrePersist(_ context.Context, cvHash proto.ConsensusValueHash) {
	cbp.lk.Lock()
	donePersisting, ok := cbp.persisting[cvHash]
	cbp.lk.Unlock()
	if !ok {
		liblog.Infof("catchup need not wait for pre-persist: cv_hash %s",
			libxdr.HashString(cvHash))
		return
	}
	<-donePersisting

	liblog.Infof("finished waiting for catchup to pre-persist: cv_hash %s",
		libxdr.HashString(cvHash))
}

func (cbp *CatchupBlockPool) CatchupBlockPoolPersist(
	ctx context.Context,
	height proto.BlockHeight,
	cvHash proto.ConsensusValueHash,
) {
	liblog.Infof("persisting catchup block: height %d; cv_hash %s",
		height, libxdr.HashString(cvHash))

	cbp.waitFinishPrePersist(ctx, cvHash)
	cbp.persist.PersistAlreadyPrePersistedCatchupBlock(ctx, height, cvHash)
}

func (cbp *CatchupBlockPool) CatchupBlockPoolFindHeight(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.CatchupBlock {
	return cbp.persist.FindCatchupBlock(ctx, height)
}
