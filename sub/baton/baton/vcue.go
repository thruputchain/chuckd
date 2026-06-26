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
	err := libxdr.UnmarshalXdrEOF(&cvc, value)
	if err != nil {
		return scpkit.InvalidValue
	}

	kind := "ballot"
	if nomination {
		kind = "nomination"
	}
	liblog.Infof(ctx, "validate value (%s) requested: slot %d; cvc %s; block_hash %s; proposer %s",
		kind, slot, protoutil.CVCDisplay(&cvc), libxdr.HashString(cvc.Block_hash),
		protoutil.VPublicShortString(cvc.Proposer))
	defer func() {
		liblog.Infof(ctx, "validate value (%s) completed: slot %d; cvc %s; block_hash %s; proposer %s; result %v",
			kind, slot, protoutil.CVCDisplay(&cvc), libxdr.HashString(cvc.Block_hash),
			protoutil.VPublicShortString(cvc.Proposer), scpkit.ValidationLevelString(level))
	}()

	x.baton.validating.Lock(protoutil.HashCVC(&cvc))
	defer x.baton.validating.Unlock(protoutil.HashCVC(&cvc))

	reply := make(chan scpkit.ValidationLevel, 1)
	go x.baton.validateCue(context.WithoutCancel(ctx), slot, &cvc, reply)
	level = <-reply
	return
}

func (b *Baton) validateCue(
	ctx context.Context,
	slot scpkit.Slot,
	cvc *proto.ConsensusValueCue,
	reply chan<- scpkit.ValidationLevel,
) {
	// validateCue reserves cancellation as a signal of externalization ahead.
	// so we make sure no other source of cancellation can be injected.
	// TODO: in a future design, context cancellation should be separate from externalization signals.
	ctx = context.WithoutCancel(ctx)

	if slot != scpkit.Slot(cvc.Block_height) {
		liblog.Errorf(ctx, "scp slot disagrees with cv height: cvc %s; slot %d; cue_height %d",
			protoutil.CVCDisplay(cvc), slot, proto.BlockHeight(cvc.Block_height))
		reply <- scpkit.InvalidValue
		return
	}

	liblog.Debugf(ctx, "validating cue: height %d; block_hash %s; cvc %s; proposer %s",
		slot, libxdr.HashString(cvc.Block_hash), protoutil.CVCDisplay(cvc),
		protoutil.VPublicShortString(cvc.Proposer))

	// check if the value was already validated
	b.lk.Lock()
	alreadyValidated := b.getValidatedBasis(cvc)
	alreadyInvalidated := b.isInvalidated(cvc)
	committedHeight := b.tipDb.Height
	b.lk.Unlock()

	// if the cv is in the future, we declare it maybe valid, which
	// allows scp to externalize it, if a quorum certificate is witnessed.
	//
	// XXX(SECURITY): we may already be committing the parent block of the validation request,
	// but this race cannot be accounted for as commit in validation are
	// not synchronizable with each other. this race should be addressed by
	// scp/protocol-level logic.
	//
	// XXX(SECURITY): adversary can force all honest nodes to declare maybe valid to a future slot.
	if cvc.Block_height > committedHeight+1 {
		liblog.Errorf(ctx, "cv in the future: cvc %s", protoutil.CVCDisplay(cvc))
		reply <- scpkit.MaybeValidValue
		return
	}

	if alreadyInvalidated {
		liblog.Errorf(ctx, "cv already invalidated: cvc %s", protoutil.CVCDisplay(cvc))
		reply <- scpkit.InvalidValue
		return
	}
	if alreadyValidated != nil {
		if protoutil.BasisAgreesWithCue(alreadyValidated, cvc) {
			liblog.Debugf(ctx, "cv already validated: cvc %s", protoutil.CVCDisplay(cvc))
			reply <- scpkit.FullyValidValue
		} else {
			liblog.Errorf(ctx, "cv already validated, but disagrees with cue: cvc %s",
				protoutil.CVCDisplay(cvc))
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
		liblog.Debugf(ctx, "cancelling validating cue: height %d; block_hash %s; cvc %s",
			slot, libxdr.HashString(cvc.Block_hash), protoutil.CVCDisplay(cvc))
		vCtxCancel()
	}()
	defer vCtxCancel()

	// fetchBasis will be aborted if externalization or commits beyond the nominated block height occur
	// BLOCKING
	basis, err := b.fetchBasis(vCtx, slot, cvc)
	if errors.Is(err, context.Canceled) {
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= proto.BlockHeight(slot))
		reply <- scpkit.MaybeValidValue
		return
	}
	must.NoError(ctx, err)

	// XXX(DoS): if basis disagrees with cue, take action regarding the source of the basis
	if !protoutil.BasisAgreesWithCue(basis, cvc) {
		liblog.Infof(ctx, "cannot validate, basis disagrees with cue")
		reply <- scpkit.InvalidValue
		return
	}

	b.validateConsensus(vCtx, slot, basis, cvc, reply)
}

func (b *Baton) fetchBasis(
	ctx context.Context,
	slot scpkit.Slot,
	cvc *proto.ConsensusValueCue,
) (*proto.ConsensusBasis, error) {

	liblog.Infof(ctx, "fetchCV starting: height=%d; block_hash=%s; cvc %s; proposer=%s",
		slot, libxdr.HashString(cvc.Block_hash), protoutil.CVCDisplay(cvc), protoutil.VPublicShortString(cvc.Proposer))

	// lookup in cvpool
	basis, err := b.handler.CVPoolLookup(ctx, *cvc)
	if errors.Is(err, context.Canceled) {
		return nil, ctx.Err()
	}
	must.NoError(ctx, err)
	if basis != nil {
		liblog.Debugf(ctx, "fetchCV found in pool: height=%d; block_hash=%s; cvc %s, proposer=%s",
			slot, libxdr.HashString(cvc.Block_hash), protoutil.CVCDisplay(cvc), protoutil.VPublicShortString(cvc.Proposer))
		return basis, nil
	}

	liblog.Debugf(ctx, "fetchCV repairing missing: height=%d; block_hash=%s; cvc %s, proposer=%s",
		slot, libxdr.HashString(cvc.Block_hash), protoutil.CVCDisplay(cvc), protoutil.VPublicShortString(cvc.Proposer))

	fetchCtx, fetchCancel := context.WithCancel(ctx)
	basisReturn := make(chan *proto.ConsensusBasis, 2)

	// try repairing the missing basis
	// this goroutine keeps on trying unless cancelled
	go func() {
		basis, err := b.handler.RepairMissingBasis(fetchCtx, cvc.Proposer, *cvc)
		if err != nil {
			must.Assert(fetchCtx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		basisReturn <- basis
	}()

	// try getting the basis from the pool, if it arrives before the repair
	// this goroutine keeps on trying unless cancelled
	go func() {
		basis, err := b.handler.CVPoolWaitFor(fetchCtx, *cvc)
		if err != nil {
			must.Assert(fetchCtx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		basisReturn <- basis
	}()

	select {
	case <-ctx.Done():
		fetchCancel()
		liblog.Errorf(ctx, "fetchCV cancelled (%v): height=%d; block_hash=%s; cvc=%s; proposer=%s",
			ctx.Err(), slot, libxdr.HashString(cvc.Block_hash), protoutil.CVCDisplay(cvc),
			protoutil.VPublicShortString(cvc.Proposer))
		return nil, ctx.Err()
	case basis, ok := <-basisReturn:
		if ok {
			fetchCancel()
			liblog.Debugf(ctx, "fetchCV found: height=%d; block_hash=%s; cvc=%s; proposer=%s",
				slot, libxdr.HashString(cvc.Block_hash), protoutil.CVCDisplay(cvc),
				protoutil.VPublicShortString(cvc.Proposer))
			b.handler.CVPoolAdd(ctx, *basis, *cvc)
			return basis, nil
		} else {
			panic("unreachable")
		}
	}
}
