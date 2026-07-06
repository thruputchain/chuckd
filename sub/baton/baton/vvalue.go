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

	kind := "ballot"
	if nomination {
		kind = "nomination"
	}
	liblog.Infof(ctx, "validate value (%s) requested: slot %d; value %s; proposer %s",
		kind, slot, protoutil.ValueDisplay(&value), protoutil.VIDString(value.Proposer))
	defer func() {
		liblog.Infof(ctx, "validate value (%s) completed: slot %d; value %s; proposer %s; result %v",
			kind, slot, protoutil.ValueDisplay(&value), protoutil.VIDString(value.Proposer),
			scpkit.ValidationLevelString(level))
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
	ctx = context.WithoutCancel(ctx)

	height := value.Proposal_header.Block_height
	if slot != scpkit.Slot(height) {
		liblog.Errorf(ctx, "scp slot disagrees with cv height: value %s; slot %d; height %d",
			protoutil.ValueDisplay(value), slot, height)
		reply <- scpkit.InvalidValue
		return
	}

	liblog.Debugf(ctx, "validating value: height %d; value %s; proposer %s",
		slot, protoutil.ValueDisplay(value), protoutil.VIDString(value.Proposer))

	b.lk.Lock()
	alreadyValidated := b.getValidatedBasis(value)
	alreadyInvalidated := b.isInvalidated(value)
	committedHeight := protoutil.TipHeight(b.tipDb)
	b.lk.Unlock()

	if height > committedHeight+1 {
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

	vCtx, vCtxCancel := context.WithCancel(ctx)
	go func() {
		b.WaitExternalizedBeyond(vCtx, height)
		liblog.Debugf(ctx, "cancelling validating value: height %d; value %s",
			slot, protoutil.ValueDisplay(value))
		vCtxCancel()
	}()
	defer vCtxCancel()

	basis, err := b.fetchBasis(vCtx, slot, value)
	if errors.Is(err, context.Canceled) {
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= height)
		reply <- scpkit.MaybeValidValue
		return
	}
	must.NoError(ctx, err)

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

	liblog.Infof(ctx, "fetchCV starting: height=%d; value %s; proposer=%s",
		slot, protoutil.ValueDisplay(value), protoutil.VIDString(value.Proposer))

	basis, err := b.handler.CVPoolLookup(ctx, *value)
	if errors.Is(err, context.Canceled) {
		return nil, ctx.Err()
	}
	must.NoError(ctx, err)
	if basis != nil {
		liblog.Debugf(ctx, "fetchCV found in pool: height=%d; value %s, proposer=%s",
			slot, protoutil.ValueDisplay(value), protoutil.VIDString(value.Proposer))
		return basis, nil
	}

	liblog.Debugf(ctx, "fetchCV repairing missing: height=%d; value %s, proposer=%s",
		slot, protoutil.ValueDisplay(value), protoutil.VIDString(value.Proposer))

	fetchCtx, fetchCancel := context.WithCancel(ctx)
	basisReturn := make(chan *proto.ConsensusBasis, 2)

	go func() {
		basis, err := b.handler.RepairMissingBasis(fetchCtx, value.Proposer, *value)
		if err != nil {
			must.Assert(fetchCtx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		basisReturn <- basis
	}()

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
			ctx.Err(), slot, protoutil.ValueDisplay(value), protoutil.VIDString(value.Proposer))
		return nil, ctx.Err()
	case basis, ok := <-basisReturn:
		if ok {
			fetchCancel()
			liblog.Debugf(ctx, "fetchCV found: height=%d; value=%s; proposer=%s",
				slot, protoutil.ValueDisplay(value), protoutil.VIDString(value.Proposer))
			b.handler.CVPoolAdd(ctx, *basis, *value)
			return basis, nil
		}
		panic("unreachable")
	}
}
