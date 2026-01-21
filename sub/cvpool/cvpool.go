package cvpool

import (
	"context"
	"sync"

	proto "github.com/groundhogchain/protocol-go"
)

type ConsensusValuePool struct {
	cond   *sync.Cond
	lk     sync.Mutex
	bottom proto.BlockHeight // height of highest externalized slot
	cv     map[proto.ConsensusValueHash]*cvBox
}

func NewConsensusValuePool(ctx context.Context) *ConsensusValuePool {
	cvp := &ConsensusValuePool{
		cv:     map[proto.ConsensusValueHash]*cvBox{},
		bottom: 0,
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
	cvp.cv[cvh] = NewCVBox(ctx, cv)
}

func (cvp *ConsensusValuePool) PurgeNotAbove(ctx context.Context, height proto.BlockHeight) {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	cvp.bottom = max(height, cvp.bottom)
	cvp.purge()
}

func (cvp *ConsensusValuePool) purge() {
	for h, cv := range cvp.cv {
		if cv.CV().Nomination.Block_height <= cvp.bottom {
			delete(cvp.cv, h)
		}
	}
}

func (cvp *ConsensusValuePool) ConsensusValuePoolLookup(
	ctx context.Context,
	h proto.ConsensusValueHash,
) CVBox {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	if cv, ok := cvp.cv[h]; ok {
		return cv
	} else {
		return nil
	}
}

func (cvp *ConsensusValuePool) ConsensusValuePoolWaitFor(
	ctx context.Context,
	hash proto.ConsensusValueHash,
) (CVBox, error) {
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
