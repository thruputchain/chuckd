package cvpool

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/sub/storage"
	proto "github.com/groundhogchain/protocol-go"
)

type ConsensusValuePool struct {
	persist storage.ConsensusValuePersistence
	cond    *sync.Cond
	lk      sync.Mutex
	bottom  proto.BlockHeight // heights at or below bottom are purged; set to last committed height
	cv      map[proto.ConsensusValueHash]*proto.ConsensusValue
}

func NewConsensusValuePool(ctx context.Context, persist storage.ConsensusValuePersistence) *ConsensusValuePool {
	cvp := &ConsensusValuePool{
		persist: persist,
		cv:      map[proto.ConsensusValueHash]*proto.ConsensusValue{},
		bottom:  0,
	}
	cvp.cond = sync.NewCond(&cvp.lk)
	var _ CVPool = cvp
	return cvp
}

func (cvp *ConsensusValuePool) ConsensusValuePoolAdd(
	ctx context.Context,
	cv proto.ConsensusValue,
	cvh proto.ConsensusValueHash,
) {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	if cv.Nomination.Block_height <= cvp.bottom {
		return
	}
	cvp.cv[cvh] = &cv
	cvp.cond.Broadcast()
	go cvp.persist.PrePersistConsensusValue(context.WithoutCancel(ctx), cv, cvh)
}

func (cvp *ConsensusValuePool) ConsensusValuePoolPersist(
	ctx context.Context,
	cv proto.ConsensusValue,
	cvh proto.ConsensusValueHash,
) {
	cvp.persist.PersistConsensusValue(ctx, cv, cvh)

	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	if cv.Nomination.Block_height <= cvp.bottom {
		return
	}
	cvp.cv[cvh] = &cv
	cvp.cond.Broadcast()
}

func (cvp *ConsensusValuePool) PurgeNotAbove(ctx context.Context, height proto.BlockHeight) {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	cvp.bottom = max(height, cvp.bottom)
	cvp.purge()
}

func (cvp *ConsensusValuePool) purge() {
	for h, cv := range cvp.cv {
		if cv.Nomination.Block_height <= cvp.bottom {
			delete(cvp.cv, h)
		}
	}
}

func (cvp *ConsensusValuePool) ConsensusValuePoolLookupHash(
	ctx context.Context,
	h proto.ConsensusValueHash,
) *proto.ConsensusValue {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()
	return cvp.cv[h]
}

func (cvp *ConsensusValuePool) ConsensusValuePoolWaitFor(
	ctx context.Context,
	hash proto.ConsensusValueHash,
) (*proto.ConsensusValue, error) {
	stop := context.AfterFunc(ctx, func() { cvp.cond.Broadcast() })
	defer stop()
	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if cv, ok := cvp.cv[hash]; ok {
				return cv, nil
			}
			cvp.cond.Wait()
		}
	}
}

// height-dependent queries

func (cvp *ConsensusValuePool) ConsensusValuePoolFindHeight(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.ConsensusValue {
	return cvp.persist.FindConsensusValue(ctx, height)
}

func (cvp *ConsensusValuePool) ConsensusValuePoolFindHighest(
	ctx context.Context,
) *proto.ConsensusValue {
	return cvp.persist.FindHighestConsensusValue(ctx)
}

func (cvp *ConsensusValuePool) ConsensusValuePoolFindAbove(
	ctx context.Context,
	above proto.BlockHeight,
) []proto.ConsensusValue {
	return storage.FindConsensusValuesAbove(ctx, cvp.persist, above)
}
