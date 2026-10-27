package baton

import (
	"context"
	"errors"

	"github.com/thruputchain/chuckd/sub/scp/scpkit"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func (x scpHandler) ValidateValue(
	ctx context.Context,
	slot scpkit.Slot,
	scpValue scpkit.Value,
	nomination bool,
) (level scpkit.ValidationLevel) {

	<-x.baton.ready

	var value proto.ConsensusValue
	err := libxdr.UnmarshalXdrEOF(&value, scpValue)
	if err != nil {
		return scpkit.InvalidValue
	}
	if value.Protocol != proto.Protocol(0) {
		return scpkit.InvalidValue
	}

	kind := "ballot"
	if nomination {
		kind = "nomination"
	}
	liblog.Debugf(ctx, "validate value (%s) requested: slot %d; value %s; proposer %s",
		kind, slot, protoutil.ValueDisplay(&value),
		protoutil.VIDString(protoutil.ValueProposer(&value)))
	defer func() {
		liblog.Debugf(ctx, "validate value (%s) completed: slot %d; value %s; proposer %s; result %v",
			kind, slot, protoutil.ValueDisplay(&value),
			protoutil.VIDString(protoutil.ValueProposer(&value)), scpkit.ValidationLevelString(level))
	}()

	x.baton.validating.Lock(protoutil.HashValue(&value))
	defer x.baton.validating.Unlock(protoutil.HashValue(&value))

	reply := make(chan scpkit.ValidationLevel, 1)
	go x.baton.validateValue(context.WithoutCancel(ctx), slot, &value, reply)
	level = <-reply
	return
}

func (b *Baton) validateValue(
	ctx context.Context,
	slot scpkit.Slot,
	value *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) {
	// validateValue reserves cancellation as a signal of externalization ahead.
	// so we make sure no other source of cancellation can be injected.
	// TODO: in a future design, context cancellation should be separate from externalization signals.
	ctx = context.WithoutCancel(ctx)

	if slot != scpkit.Slot(protoutil.ValueHeight(value)) {
		liblog.Errorf(ctx, "scp slot disagrees with cv height: value %s; slot %d; height %d",
			protoutil.ValueDisplay(value), slot, protoutil.ValueHeight(value))
		reply <- scpkit.InvalidValue
		return
	}

	liblog.Debugf(ctx, "validating value: height %d; value %s; proposer %s",
		slot, protoutil.ValueDisplay(value),
		protoutil.VIDString(protoutil.ValueProposer(value)))

	// check if the value was already validated
	b.lk.Lock()
	alreadyValidated := b.getValidatedBasis(value)
	alreadyInvalidated := b.isInvalidated(value)
	committedHeight := protoutil.ValueHeight(&b.tipDb.Value)
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
	if protoutil.ValueHeight(value) > committedHeight+1 {
		liblog.Errorf(ctx, "value in the future: value %s", protoutil.ValueDisplay(value))
		reply <- scpkit.MaybeValidValue
		return
	}

	if alreadyInvalidated {
		liblog.Errorf(ctx, "value already invalidated: value %s", protoutil.ValueDisplay(value))
		reply <- scpkit.InvalidValue
		return
	}
	if alreadyValidated != nil {
		if protoutil.BasisAgreesWithValue(alreadyValidated, value) {
			liblog.Debugf(ctx, "basis already validated: value %s", protoutil.ValueDisplay(value))
			reply <- scpkit.FullyValidValue
		} else {
			liblog.Errorf(ctx, "basis already validated, but disagrees with value: value %s",
				protoutil.ValueDisplay(value))
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
		liblog.Debugf(ctx, "cancelling validating value: height %d; value %s",
			slot, protoutil.ValueDisplay(value))
		vCtxCancel()
	}()
	defer vCtxCancel()

	// fetchBasis will be aborted if externalization or commits beyond the nominated block height occur
	// BLOCKING
	basis, err := b.fetchBasis(vCtx, slot, value)
	if errors.Is(err, context.Canceled) {
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= proto.BlockHeight(slot))
		reply <- scpkit.MaybeValidValue
		return
	}
	must.NoError(ctx, err)

	// XXX(DoS): if basis disagrees with value, take action regarding the source of the basis
	if !protoutil.BasisAgreesWithValue(basis, value) {
		liblog.Infof(ctx, "cannot validate, basis disagrees with value")
		reply <- scpkit.InvalidValue
		return
	}

	b.validateBasis(vCtx, slot, basis, value, reply)
}

func (b *Baton) fetchBasis(
	ctx context.Context,
	slot scpkit.Slot,
	value *proto.ConsensusValue,
) (*proto.ConsensusBasis, error) {

	liblog.Debugf(ctx, "fetchCV starting: height=%d; value %s; proposer=%s",
		slot, protoutil.ValueDisplay(value), protoutil.VIDString(protoutil.ValueProposer(value)))

	// lookup in cvpool
	basis, err := b.handler.CVPoolLookup(ctx, *value)
	if errors.Is(err, context.Canceled) {
		return nil, ctx.Err()
	}
	must.NoError(ctx, err)
	if basis != nil {
		liblog.Debugf(ctx, "fetchCV found in pool: height=%d; value %s, proposer=%s",
			slot, protoutil.ValueDisplay(value), protoutil.VIDString(protoutil.ValueProposer(value)))
		return basis, nil
	}

	liblog.Debugf(ctx, "fetchCV repairing missing: height=%d; value %s, proposer=%s",
		slot, protoutil.ValueDisplay(value), protoutil.VIDString(protoutil.ValueProposer(value)))

	fetchCtx, fetchCancel := context.WithCancel(ctx)
	basisReturn := make(chan *proto.ConsensusBasis, 2)

	// try repairing the missing basis
	// this goroutine keeps on trying unless cancelled
	go func() {
		basis, err := b.handler.RepairMissingBasis(fetchCtx, protoutil.ValueProposer(value), *value)
		if err != nil {
			must.Assert(fetchCtx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		basisReturn <- basis
	}()

	// try getting the basis from the pool, if it arrives before the repair
	// this goroutine keeps on trying unless cancelled
	go func() {
		basis, err := b.handler.CVPoolWaitFor(fetchCtx, *value)
		if err != nil {
			must.Assert(fetchCtx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		basisReturn <- basis
	}()

	select {
	case <-ctx.Done():
		fetchCancel()
		liblog.Errorf(ctx, "fetchCV cancelled (%v): height=%d; value=%s; proposer=%s",
			ctx.Err(), slot, protoutil.ValueDisplay(value),
			protoutil.VIDString(protoutil.ValueProposer(value)))
		return nil, ctx.Err()
	case basis, ok := <-basisReturn:
		if ok {
			fetchCancel()
			liblog.Debugf(ctx, "fetchCV found: height=%d; value=%s; proposer=%s",
				slot, protoutil.ValueDisplay(value),
				protoutil.VIDString(protoutil.ValueProposer(value)))
			b.handler.CVPoolAdd(ctx, *basis, *value)
			return basis, nil
		} else {
			panic("unreachable")
		}
	}
}
