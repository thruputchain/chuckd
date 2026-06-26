package catchup

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type CatchupBlockPool struct {
	persist    storage.CatchupBlockPersistence
	lk         sync.Mutex
	persisting map[protoutil.ValueHash]chan struct{}
}

func NewCatchupBlockPool(ctx context.Context, persist storage.CatchupBlockPersistence) *CatchupBlockPool {
	cbp := &CatchupBlockPool{
		persist:    persist,
		persisting: map[protoutil.ValueHash]chan struct{}{},
	}
	var _ CBPool = cbp
	return cbp
}

func (cbp *CatchupBlockPool) CBPoolAdd(
	ctx context.Context,
	cb proto.CatchupBlock,
	value proto.ConsensusValue,
) error {
	cbp.lk.Lock()
	defer cbp.lk.Unlock()

	vhash := protoutil.HashValue(&value)

	liblog.Infof(ctx, "adding catchup block to pool: height %d; cvc %s; proposer %s",
		cb.Consensus_basis.Nomination.Block_height, vhash.Display(),
		protoutil.VPublicDisplay(cb.Consensus_basis.Nomination.Proposer))

	if _, alreadyPersisting := cbp.persisting[vhash]; alreadyPersisting {
		liblog.Infof(ctx, "catchup block already persisting: height %d; cvc %s; proposer %s",
			cb.Consensus_basis.Nomination.Block_height, vhash.Display(),
			protoutil.VPublicDisplay(cb.Consensus_basis.Nomination.Proposer))
		return nil
	}

	donePersisting := make(chan struct{})
	cbp.persisting[vhash] = donePersisting

	go func() {
		err := cbp.persist.PrePersistCatchupBlock(context.WithoutCancel(ctx), cb, vhash)
		must.NoError(ctx, err)

		close(donePersisting)

		liblog.Debugf(ctx, "finished pre-persisting catchup block: height %d; cvc %s; proposer %s",
			cb.Consensus_basis.Nomination.Block_height, vhash.Display(),
			protoutil.VPublicDisplay(cb.Consensus_basis.Nomination.Proposer))

		cbp.lk.Lock()
		defer cbp.lk.Unlock()
		delete(cbp.persisting, vhash)
	}()

	return nil
}

func (cbp *CatchupBlockPool) waitFinishPrePersist(ctx context.Context, value proto.ConsensusValue) error {
	vhash := protoutil.HashValue(&value)

	cbp.lk.Lock()
	donePersisting, ok := cbp.persisting[vhash]
	cbp.lk.Unlock()
	if !ok {
		liblog.Infof(ctx, "catchup need not wait for pre-persist: cvc %s",
			vhash.Display())
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-donePersisting:
		liblog.Infof(ctx, "finished waiting for catchup to pre-persist: cvc %s",
			vhash.Display())
		return nil
	}
}

func (cbp *CatchupBlockPool) CBPoolWaitPrePersist(
	ctx context.Context,
	value proto.ConsensusValue,
) error {
	liblog.Infof(ctx, "waiting for pre-persisting of catchup block: height %d; cvc %s",
		value.Block_height, protoutil.ValueDisplay(&value))

	// NOTE: we have no way of knowing currently if the block had initiated pre-persist at all.
	err := cbp.waitFinishPrePersist(ctx, value)
	if err != nil {
		return err
	}

	return nil
}

func (cbp *CatchupBlockPool) CBPoolWaitPersist(
	ctx context.Context,
	value proto.ConsensusValue,
) error {
	liblog.Infof(ctx, "waiting for persisting of catchup block: height %d; cvc %s",
		value.Block_height, protoutil.ValueDisplay(&value))

	err := cbp.waitFinishPrePersist(ctx, value)
	if err != nil {
		return err
	}

	return cbp.persist.PersistAlreadyPrePersistedCatchupBlock(ctx, value.Block_height, protoutil.HashValue(&value))
}

// CBPoolLookup finds persisted or pre-persisted catchup blocks with given cv hash.
// Blocks we have nominated but are not committed will be found in pre-persist.
func (cbp *CatchupBlockPool) CBPoolLookup(
	ctx context.Context,
	value proto.ConsensusValue,
) (*proto.CatchupBlock, error) {
	err := cbp.waitFinishPrePersist(ctx, value)
	if err != nil {
		return nil, err
	}

	cb, err := cbp.persist.LookupCatchupBlock(ctx, protoutil.HashValue(&value))
	if err != nil {
		return nil, err
	}
	if cb == nil {
		return nil, nil
	}

	if cb.Consensus_basis.Nomination.Block_height != value.Block_height {
		return nil, nil
	}
	return cb, nil
}

func (cbp *CatchupBlockPool) CBPoolFindCommitted(
	ctx context.Context,
	height proto.BlockHeight,
) (*proto.CatchupBlock, error) {
	return cbp.persist.FindCatchupBlock(ctx, height)
}
