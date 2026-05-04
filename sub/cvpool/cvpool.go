package cvpool

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type ConsensusValuePool struct {
	persist storage.ConsensusValuePersistence
	cond    *sync.Cond
	lk      sync.Mutex
	bottom  proto.BlockHeight // heights at or below bottom are purged; set to last committed height
	cv      map[proto.ConsensusValueHash]*proto.ConsensusValue
}

func NewCVPool(ctx context.Context, persist storage.ConsensusValuePersistence) *ConsensusValuePool {
	cvp := &ConsensusValuePool{
		persist: persist,
		cv:      map[proto.ConsensusValueHash]*proto.ConsensusValue{},
		bottom:  0,
	}
	cvp.cond = sync.NewCond(&cvp.lk)
	var _ CVPool = cvp
	return cvp
}

func (cvp *ConsensusValuePool) CVPoolAdd(
	ctx context.Context,
	cv proto.ConsensusValue,
	cue proto.ConsensusValueCue,
) {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	if cv.Nomination.Block_height <= cvp.bottom {
		return
	}
	cvp.cv[cue.Cv_hash] = &cv
	cvp.cond.Broadcast()
	go cvp.persist.PrePersistCV(context.WithoutCancel(ctx), cv, cue.Cv_hash)
}

func (cvp *ConsensusValuePool) CVPoolCommit(
	ctx context.Context,
	cv proto.ConsensusValue,
	cue proto.ConsensusValueCue,
) {
	cvp.persist.PersistCV(ctx, cv, cue.Cv_hash)

	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	if cv.Nomination.Block_height <= cvp.bottom {
		return
	}
	cvp.cv[cue.Cv_hash] = &cv
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

func (cvp *ConsensusValuePool) CVPoolLookup(
	ctx context.Context,
	cue proto.ConsensusValueCue,
) *proto.ConsensusValue {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()
	cv, ok := cvp.cv[cue.Cv_hash]
	if ok && protoutil.CVAgreesWithCue(cv, &cue) {
		return cv
	}
	return nil
}

func (cvp *ConsensusValuePool) CVPoolWaitFor(
	ctx context.Context,
	cue proto.ConsensusValueCue,
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
			if cv, ok := cvp.cv[cue.Cv_hash]; ok && protoutil.CVAgreesWithCue(cv, &cue) {
				return cv, nil
			}
			cvp.cond.Wait()
		}
	}
}

// height-dependent queries

func (cvp *ConsensusValuePool) CVPoolFind(
	ctx context.Context,
	cue proto.ConsensusValueCue,
) *proto.ConsensusValue {
	cv := cvp.persist.FindCV(ctx, cue.Block_height, cue.Cv_hash)
	if cv != nil && protoutil.CVAgreesWithCue(cv, &cue) {
		return cv
	}
	return nil
}

func (cvp *ConsensusValuePool) CVPoolFindCommitted(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.ConsensusValue {
	return cvp.persist.FindCommittedCV(ctx, height)
}

func (cvp *ConsensusValuePool) CVPoolFindHighestCommitted(
	ctx context.Context,
) *proto.ConsensusValue {
	return cvp.persist.FindHighestCommittedCV(ctx)
}

func (cvp *ConsensusValuePool) CVPoolFindCommittedAbove(
	ctx context.Context,
	above proto.BlockHeight,
) []proto.ConsensusValue {
	return storage.FindCommittedConsensusValuesAbove(ctx, cvp.persist, above)
}
