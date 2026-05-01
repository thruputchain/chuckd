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

	var cvc proto.ConsensusValueCue
	err := libxdr.UnmarshalXdr(&cvc, value)
	if err != nil {
		return scpkit.InvalidValue
	}

	kind := "ballot"
	if nomination {
		kind = "nomination"
	}
	liblog.Infof(ctx, "validate value (%s) requested: slot %d; cv_hash %s; block_hash %s; proposer %s",
		kind, slot, libxdr.HashString(cvc.Cv_hash), libxdr.HashString(cvc.Block_hash),
		protoutil.VPublicShortString(cvc.Proposer))
	defer func() {
		liblog.Infof(ctx, "validate value (%s) completed: slot %d; cv_hash %s; block_hash %s; proposer %s; result %v",
			kind, slot, libxdr.HashString(cvc.Cv_hash), libxdr.HashString(cvc.Block_hash),
			protoutil.VPublicShortString(cvc.Proposer), scpkit.ValidationLevelString(level))
	}()

	x.baton.validating.Lock(cvc.Cv_hash)
	defer x.baton.validating.Unlock(cvc.Cv_hash)

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
		liblog.Errorf(ctx, "scp slot disagrees with cv cue: cv_hash %s; slot %d; cue_height %d",
			libxdr.HashString(cvc.Cv_hash), slot, proto.BlockHeight(cvc.Block_height))
		reply <- scpkit.InvalidValue
		return
	}

	liblog.Debugf(ctx, "validating cue: height %d; block_hash %s; cv_hash %s; proposer %s",
		slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash),
		protoutil.VPublicShortString(cvc.Proposer))

	// check if the value was already validated
	b.lock()
	alreadyValidated := b.getValidatedCV(cvc.Cv_hash, cvc.Block_height)
	alreadyInvalidated := b.isInvalidated(cvc.Cv_hash, cvc.Block_height)
	committedHeight := b.tipDb.Height
	b.unlock()

	// if the cv is in the future, we declare it maybe valid, which
	// allows scp to externalize it, if a quorum certificate is witnessed.
	// XXX(SECURITY): adversary can force all honest nodes to declare maybe valid to a future slot.
	if cvc.Block_height > committedHeight+1 {
		liblog.Errorf(ctx, "cv in the future: cv_hash %s", libxdr.HashString(cvc.Cv_hash))
		reply <- scpkit.MaybeValidValue
		return
	}

	if alreadyInvalidated {
		liblog.Errorf(ctx, "cv already invalidated: cv_hash %s", libxdr.HashString(cvc.Cv_hash))
		reply <- scpkit.InvalidValue
		return
	}
	if alreadyValidated != nil {
		if b.ConsensusValueAgreesWithCue(alreadyValidated, cvc) {
			liblog.Debugf(ctx, "cv already validated: cv_hash %s", libxdr.HashString(cvc.Cv_hash))
			reply <- scpkit.FullyValidValue
		} else {
			liblog.Errorf(ctx, "cv already validated, but disagrees with cue: cv_hash %s",
				libxdr.HashString(cvc.Cv_hash))
			reply <- scpkit.InvalidValue
		}
		return
	}

	// the goroutine below will abort following expensive fetch/repair operations if the cv gets
	// externalized in the meantime.
	// (abort condition) this context will be cancelled upon newer externalizations or commits
	vCtx, vCtxCancel := context.WithCancel(ctx)
	go func() {
		// waiter is cancelled whenever validation completes or externalization occurs
		b.WaitExternalizedBeyond(vCtx, proto.BlockHeight(slot)) // cancel ctx if tip or ext moves
		liblog.Debugf(ctx, "cancelling validating cue: height %d; block_hash %s; cv_hash %s",
			slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash))
		vCtxCancel()
	}()
	defer vCtxCancel()

	// fetchCV will be aborted if externalization or commits beyond the nominated block height occur
	// BLOCKING
	cv, err := b.fetchCV(vCtx, slot, cvc)
	if err != nil {
		must.Assert(ctx, errors.Is(err, context.Canceled))
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= proto.BlockHeight(slot))
		reply <- scpkit.MaybeValidValue
		return
	}

	// XXX(DoS): if cv disagrees with cue, take action regarding the source of the cv
	if !b.ConsensusValueAgreesWithCue(cv, cvc) {
		liblog.Infof(ctx, "cannot validate, consensus value disagrees with cue")
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

	liblog.Infof(ctx, "fetchCV starting: height=%d; block_hash=%s; cv_hash=%s; proposer=%s",
		slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash), protoutil.VPublicShortString(cvc.Proposer))

	// lookup in cvpool
	cv := b.handler.CVPoolLookup(ctx, cvc.Block_height, cvc.Cv_hash)
	if cv != nil {
		liblog.Debugf(ctx, "fetchCV found in pool: height=%d; block_hash=%s; cv_hash=%s, proposer=%s",
			slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash), protoutil.VPublicShortString(cvc.Proposer))
		return cv, nil
	}

	liblog.Debugf(ctx, "fetchCV repairing missing: height=%d; block_hash=%s; cv_hash=%s, proposer=%s",
		slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash), protoutil.VPublicShortString(cvc.Proposer))

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
		cv, err := b.handler.CVPoolWaitFor(ctx, cvc.Block_height, cvc.Cv_hash)
		if err != nil {
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		cvReturn <- cv
	}()

	select {
	case <-ctx.Done():
		liblog.Errorf(ctx, "fetchCV cancelled (%v): height=%d; block_hash=%s; cv_hash=%s; proposer=%s",
			ctx.Err(), slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash),
			protoutil.VPublicShortString(cvc.Proposer))
		return nil, ctx.Err()
	case cv, ok := <-cvReturn:
		if ok {
			liblog.Debugf(ctx, "fetchCV found: height=%d; block_hash=%s; cv_hash=%s; proposer=%s",
				slot, libxdr.HashString(cvc.Block_hash), libxdr.HashString(cvc.Cv_hash),
				protoutil.VPublicShortString(cvc.Proposer))
			b.handler.CVPoolAdd(ctx, *cv, cvc.Cv_hash)
			return cv, nil
		} else {
			panic("unreachable")
		}
	}
}
