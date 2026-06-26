package cvpool

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type ConsensusValuePool struct {
	persist storage.ConsensusValuePersistence
	cond    *sync.Cond
	lk      sync.Mutex
	bottom  proto.BlockHeight // heights at or below bottom are purged; set to last committed height
	basis   map[protoutil.CVCHash]*proto.ConsensusBasis
}

func NewCVPool(ctx context.Context, persist storage.ConsensusValuePersistence) *ConsensusValuePool {
	cvp := &ConsensusValuePool{
		persist: persist,
		basis:   map[protoutil.CVCHash]*proto.ConsensusBasis{},
		bottom:  0,
	}
	cvp.cond = sync.NewCond(&cvp.lk)
	var _ CVPool = cvp
	return cvp
}

func (cvp *ConsensusValuePool) CVPoolAdd(
	ctx context.Context,
	basis proto.ConsensusBasis,
	cue proto.ConsensusValueCue,
) error {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	if basis.Nomination.Block_height <= cvp.bottom {
		return nil
	}
	cvch := protoutil.HashCVC(&cue)
	cvp.basis[cvch] = &basis
	cvp.cond.Broadcast()
	go func() {
		err := cvp.persist.PrePersistCV(context.WithoutCancel(ctx), basis, cvch)
		must.NoError(ctx, err)
	}()
	return nil
}

func (cvp *ConsensusValuePool) CVPoolCommit(
	ctx context.Context,
	basis proto.ConsensusBasis,
	cue proto.ConsensusValueCue,
) error {
	cvch := protoutil.HashCVC(&cue)
	err := cvp.persist.PersistCV(context.WithoutCancel(ctx), basis, cvch)
	must.NoError(ctx, err)

	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	if basis.Nomination.Block_height <= cvp.bottom {
		return nil
	}
	cvp.basis[cvch] = &basis
	cvp.cond.Broadcast()
	return nil
}

func (cvp *ConsensusValuePool) PurgeNotAbove(ctx context.Context, height proto.BlockHeight) {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()

	cvp.bottom = max(height, cvp.bottom)
	cvp.purge()
}

func (cvp *ConsensusValuePool) purge() {
	for h, basis := range cvp.basis {
		if basis.Nomination.Block_height <= cvp.bottom {
			delete(cvp.basis, h)
		}
	}
}

func (cvp *ConsensusValuePool) CVPoolLookup(
	ctx context.Context,
	cue proto.ConsensusValueCue,
) (*proto.ConsensusBasis, error) {
	cvp.lk.Lock()
	defer cvp.lk.Unlock()
	cvch := protoutil.HashCVC(&cue)
	basis, ok := cvp.basis[cvch]
	if ok && protoutil.BasisAgreesWithCue(basis, &cue) {
		return basis, nil
	}
	return nil, nil
}

func (cvp *ConsensusValuePool) CVPoolWaitFor(
	ctx context.Context,
	cue proto.ConsensusValueCue,
) (*proto.ConsensusBasis, error) {
	cvch := protoutil.HashCVC(&cue)

	stop := context.AfterFunc(ctx, func() {
		cvp.lk.Lock()
		defer cvp.lk.Unlock()
		cvp.cond.Broadcast()
	})
	defer stop()

	cvp.lk.Lock()
	defer cvp.lk.Unlock()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if basis, ok := cvp.basis[cvch]; ok && protoutil.BasisAgreesWithCue(basis, &cue) {
				return basis, nil
			}
			cvp.cond.Wait()
		}
	}
}

// height-dependent queries

func (cvp *ConsensusValuePool) CVPoolFind(
	ctx context.Context,
	cue proto.ConsensusValueCue,
) (*proto.ConsensusBasis, error) {
	cvch := protoutil.HashCVC(&cue)
	basis, err := cvp.persist.FindCV(ctx, cue.Block_height, cvch)
	if err != nil {
		return nil, err
	}
	if basis != nil && protoutil.BasisAgreesWithCue(basis, &cue) {
		return basis, nil
	}
	return nil, nil
}

func (cvp *ConsensusValuePool) CVPoolFindCommitted(
	ctx context.Context,
	height proto.BlockHeight,
) (*proto.ConsensusBasis, error) {
	return cvp.persist.FindCommittedCV(ctx, height)
}

func (cvp *ConsensusValuePool) CVPoolFindHighestCommitted(
	ctx context.Context,
) (*proto.ConsensusBasis, error) {
	return cvp.persist.FindHighestCommittedCV(ctx)
}

func (cvp *ConsensusValuePool) CVPoolFindCommittedAbove(
	ctx context.Context,
	above proto.BlockHeight,
) ([]proto.ConsensusBasis, error) {
	return storage.FindCommittedConsensusValuesAbove(ctx, cvp.persist, above)
}
