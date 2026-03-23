package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (x scpHandler) ValidateCue(
	ctx context.Context,
	slot scpkit.Slot,
	value scpkit.Value,
	nomination bool,
) (level scpkit.ValidationLevel) {

	<-x.baton.ready

	kind := "ballot"
	if nomination {
		kind = "nomination"
	}
	liblog.Infof("validate value (%s) for slot %d requested", kind, slot)
	defer func() {
		liblog.Infof("validate value (%s) for slot %d completed with %v", kind, slot, scpkit.ValidationLevelString(level))
	}()

	var cvc proto.ConsensusValueCue
	err := libxdr.UnmarshalXdr(&cvc, value)
	if err != nil {
		return scpkit.InvalidValue
	}

	reply := make(chan scpkit.ValidationLevel, 1)
	go x.baton.validateCue(ctx, slot, &cvc, reply)
	level = <-reply
	return
}

func (b *Baton) validateCue(
	ctx context.Context,
	slot scpkit.Slot,
	cvc *proto.ConsensusValueCue,
	reply chan<- scpkit.ValidationLevel,
) {

	if slot != scpkit.Slot(cvc.Block_height) {
		reply <- scpkit.InvalidValue
		return
	}

	liblog.Infof("validating cue: height %d; block_hash %s; cv_hash %s; proposer %s",
		slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash),
		protoutil.VPublicShortString(cvc.Proposer))

	// check if the value was already validated
	b.lock()
	alreadyValidated := b.getValidatedConsensusFromCVH(cvc.Cv_hash)
	alreadyInvalidated := b.isInvalidatedCVH(cvc.Cv_hash)
	b.unlock()

	if alreadyInvalidated {
		liblog.Errorf("cv already invalidated: cv_hash %s", libxdr.HashString(cvc.Cv_hash))
		reply <- scpkit.InvalidValue
		return
	}
	if alreadyValidated != nil {
		if b.ConsensusValueAgreesWithCue(alreadyValidated, cvc) {
			liblog.Infof("cv already validated: cv_hash %s", libxdr.HashString(cvc.Cv_hash))
			reply <- scpkit.FullyValidValue
		} else {
			liblog.Errorf("cv already validated, but disagrees with cue: cv_hash %s",
				libxdr.HashString(cvc.Cv_hash))
			reply <- scpkit.InvalidValue
		}
		return
	}

	// (abort condition) this context will be cancelled upon newer externalizations or commits
	vCtx, vCtxCancel := context.WithCancel(ctx)
	go func() {
		// waiter is cancelled whenever validation completes or externalization occurs
		b.WaitExternalizedBeyond(vCtx, proto.BlockHeight(slot)) // cancel ctx if tip or ext moves
		liblog.Infof("cancelling validating cue at height %d, block hash %s, consensus hash %s",
			slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash))
		vCtxCancel()
	}()
	defer vCtxCancel()

	// fetchCV will be aborted if externalization or commits beyond the nominated block height occur
	// BLOCKING
	cv, err := b.fetchCV(vCtx, slot, cvc)
	if err != nil {
		must.Assert(ctx, errors.Is(err, context.Canceled))
		b.validateExternalizedCue(ctx, slot, cvc, reply)
		return
	}

	if !b.ConsensusValueAgreesWithCue(cv, cvc) {
		liblog.Infof("cannot validate, consensus value disagrees with cue")
		reply <- scpkit.InvalidValue
		return
	}

	b.validateConsensus(vCtx, slot, cv, cvc, reply)
}

func (b *Baton) fetchCV(
	ctx context.Context,
	slot scpkit.Slot,
	cvc *proto.ConsensusValueCue,
) (*proto.ConsensusValue, error) {

	liblog.Infof("fetching consensus value at height %d, block hash %s, consensus hash %s",
		slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash))

	// lookup in cvpool
	cv := b.handler.ConsensusValuePoolLookupHash(ctx, cvc.Cv_hash)
	if cv != nil {
		return cv, nil
	}

	liblog.Debugf("repairing missing consensus value for slot %d", slot)

	cvReturn := make(chan *proto.ConsensusValue, 2)

	// try repairing the missing cv
	// this goroutine keeps on trying unless cancelled
	go func() {
		cv, err := b.handler.RepairMissingCV(ctx, cvc.Proposer, cvc.Cv_hash, proto.BlockHeight(slot))
		if err != nil {
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		cvReturn <- cv
	}()

	// try getting the cv from the pool, if it arrives before the repair
	// this goroutine keeps on trying unless cancelled
	go func() {
		cv, err := b.handler.ConsensusValuePoolWaitFor(ctx, cvc.Cv_hash)
		if err != nil {
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		cvReturn <- cv
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case cv, ok := <-cvReturn:
		if ok {
			b.handler.ConsensusValuePoolAdd(ctx, *cv, cvc.Cv_hash)
			return cv, nil
		}
		return nil, context.Canceled
	}
}
