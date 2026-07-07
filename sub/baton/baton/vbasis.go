package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) validateBasis(
	ctx context.Context,
	slot scpkit.Slot,
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) {

	liblog.Infof(ctx, "validating cv: height %d; value %s; proposer %s",
		slot, protoutil.ValueDisplay(value),
		protoutil.VIDString(protoutil.ValueProposer(value)))

	if b.tryValidateBasisFast(ctx, slot, basis, value, reply) {
		return
	}

	// (commence condition) wait for the block below the validation request to be reached and committed

	var dbTip *proto.DbTip

	var err error
	dbTip, err = b.WaitCommittedAt(ctx, protoutil.ValueHeight(&basis.Value)-1)
	if err != nil { // context cancelled
		switch {
		case errors.Is(err, ErrTipAlreadyAhead): // from WaitCommittedAt
			must.Assert(ctx, b.HighestCommitted() >= protoutil.ValueHeight(&basis.Value))
			b.validateCommittedValue(ctx, slot, basis, value, reply)
			return
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded): // from WaitExternalizedBeyond
			highestExternalized := b.HighestCommittedOrExternalized()
			liblog.Infof(ctx, "skipping validation due to externalizations ahead: height: %d; highest_externalized: %d",
				protoutil.ValueHeight(&basis.Value), highestExternalized)
			// assert highest externalized is at the basis height or above
			must.Assert(ctx, highestExternalized >= protoutil.ValueHeight(&basis.Value))
			reply <- scpkit.MaybeValidValue
			return
		}
		panic("unreachable")
	}

	must.Assert(ctx, protoutil.ValueHeight(&dbTip.Value)+1 == protoutil.ValueHeight(&basis.Value))

	// validate chaining with previous consensus

	tipValueHash := protoutil.HashValueToProto(&dbTip.Value)
	if value.Prev_value_hash != tipValueHash {
		liblog.Errorf(ctx, "cannot validate, previous value hash mismatch, proposal %v vs ours %v",
			libxdr.HashString(value.Prev_value_hash),
			libxdr.HashString(tipValueHash))
		reply <- scpkit.InvalidValue
		return
	}

	// validate timestamp

	prevStamp := protoutil.TimeFromTimepoint(protoutil.ValueTimepoint(&dbTip.Value))
	newStamp := protoutil.TimeFromTimepoint(protoutil.ValueTimepoint(&basis.Value))
	if !prevStamp.Before(newStamp) {
		liblog.Errorf(ctx, "cannot validate, timestamp not increasing")
		reply <- scpkit.InvalidValue
		return
	}
	now := b.handler.Now()
	if !newStamp.Before(now) {
		liblog.Errorf(ctx, "cannot validate, timestamp in the future (stamp %s, now %s)",
			libtime.FmtTime(newStamp), libtime.FmtTime(now))
		reply <- scpkit.InvalidValue
		return
	}

	// validate proposal

	// check if block previous invalidated to prevent expensive validation
	b.lk.Lock()
	alreadyInvalidated := b.isInvalidated(value)
	b.lk.Unlock()
	if alreadyInvalidated {
		reply <- scpkit.InvalidValue
		return
	}

	validatedBlockID, err := b.validateProposal(ctx, *dbTip, basis, value)
	if errors.Is(err, batonkit.ErrNotValid) {
		// cache invalidation to prevent dos
		b.lk.Lock()
		b.setInvalidated(basis, value)
		b.lk.Unlock()

		reply <- scpkit.InvalidValue
		return
	}
	if err != nil {
		// err can only ever be context cancelled or deadline exceeded.
		// it occurs only if scp has externalized newer blocks
		must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= protoutil.ValueHeight(&basis.Value))
		reply <- scpkit.MaybeValidValue
		return
	}

	// cache validated blocks
	liblog.Debugf(ctx, "caching validation for consensus %s",
		protoutil.ValueDisplay(value))
	b.lk.Lock()
	b.setValidated(basis, value, validatedBlockID)
	b.lk.Unlock()

	reply <- scpkit.FullyValidValue
}

// tryValidateBasisFast returns true if it was able to make a judgement and send a reply.
func (b *Baton) tryValidateBasisFast(
	ctx context.Context,
	slot scpkit.Slot,
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) bool {

	if protoutil.ValueHeight(&basis.Value) != proto.BlockHeight(slot) {
		reply <- scpkit.InvalidValue
		return true
	}

	b.lk.Lock()
	dbTipOnRequest := b.tipDb
	alreadyValidated := b.getValidatedBlockID(value)
	b.lk.Unlock()

	if !protoutil.IsValidBasis(basis) {
		liblog.Infof(ctx, "cannot validate, basis not valid")
		reply <- scpkit.InvalidValue
		return true
	}

	if protoutil.ValueHeight(&basis.Value) <= protoutil.ValueHeight(&dbTipOnRequest.Value) {
		b.validateCommittedValue(ctx, slot, basis, value, reply)
		return true
	}

	if alreadyValidated != nil {
		reply <- scpkit.FullyValidValue
		return true
	}

	return false
}
