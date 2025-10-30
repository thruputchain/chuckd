package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/xdrutil"
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
	liblog.Infof("baton validate %s value for slot %d received", kind, slot)
	defer func() {
		liblog.Infof("baton validate %s value for slot %d completed with %v", kind, slot, scpkit.ValidationLevelString(level))
	}()

	var cvc proto.ConsensusValueCue
	err := xdrutil.UnmarshalXdr(&cvc, value)
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

	liblog.Infof("validating block at height %d with block hash %s and consensus value hash %s",
		slot,
		form.Display(cvc.Block_hash),
		form.Display(cvc.Cv_hash),
	)

	// check if the value was already validated
	b.lock()
	alreadyValidated := b.getValidatedCVH(cvc.Cv_hash)
	alreadyInvalidated := b.isInvalidatedCVH(cvc.Cv_hash)
	b.unlock()

	if alreadyInvalidated {
		reply <- scpkit.InvalidValue
		return
	}
	if alreadyValidated != nil {
		// XXX: check that the other values in the cue are correct
		reply <- scpkit.FullyValidValue
		return
	}

	// (abort condition) this context will be cancelled upon newer externalizations or commits
	vCtx, vCtxCancel := context.WithCancel(ctx)
	go func() {
		// waiter is cancelled whenever validation completes or externalization occurs
		b.WaitExternalizedBeyond(vCtx, proto.BlockHeight(slot)) // cancel ctx if tip or ext moves
		vCtxCancel()
	}()
	defer vCtxCancel()

	// fetchCV will be aborted if externalization or commits beyond the nominated block height occur
	// BLOCKING
	cv, err := b.fetchCV(vCtx, cvc)
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

	b.validateConsensus(vCtx, slot, cv, reply)
}

func (b *Baton) fetchCV(ctx context.Context, cvc *proto.ConsensusValueCue) (*proto.ConsensusValue, error) {

	// lookup in cvpool
	cv := b.handler.LookupConsensusValue(ctx, cvc.Cv_hash)
	if cv != nil {
		return cv, nil
	}

	cvReturn := make(chan *proto.ConsensusValue, 2)

	// try repairing the missing cv
	// this goroutine keeps on trying unless cancelled
	go func() {
		cv, err := b.handler.RepairMissingCV(ctx, cvc.Proposer, cvc.Cv_hash)
		if err != nil {
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		cvReturn <- cv
	}()

	// try getting the cv from the pool, if it arrives before the repair
	// this goroutine keeps on trying unless cancelled
	go func() {
		cv, err := b.handler.WaitForConsensusValue(ctx, cvc.Cv_hash)
		if err != nil {
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		cvReturn <- cv
	}()

	cv, ok := <-cvReturn
	if ok {
		return cv, nil
	}
	return nil, context.Canceled
}
