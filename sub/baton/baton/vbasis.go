package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
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

	liblog.Infof(ctx, "validating cv: height %d; block_hash %s; value %s; proposer %s",
		slot, libxdr.HashString(basis.Nomination.Block_hash), protoutil.ValueDisplay(value),
		protoutil.VPublicDisplay(value.Proposer))

	if b.tryValidateBasisFast(ctx, slot, basis, value, reply) {
		return
	}

	// (commence condition) wait for the block below the validation request to be reached and committed

	var dbTip *proto.DbTip
	var extTip *externalizedBasis

	var err error
	dbTip, extTip, err = b.WaitCommittedAt(ctx, basis.Nomination.Block_height-1)
	if err != nil { // context cancelled
		switch {
		case errors.Is(err, ErrTipAlreadyAhead): // from WaitCommittedAt
			must.Assert(ctx, b.HighestCommitted() >= basis.Nomination.Block_height)
			b.validateCommittedConsensus(ctx, slot, basis, value, reply)
			return
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded): // from WaitExternalizedBeyond
			highestExternalized := b.HighestCommittedOrExternalized()
			liblog.Infof(ctx, "skipping validation due to externalizations ahead: height: %d; highest_externalized: %d",
				basis.Nomination.Block_height, highestExternalized)
			// assert highest externalized is at basis.Nomination.Block_height or above
			must.Assert(ctx, highestExternalized >= basis.Nomination.Block_height)
			reply <- scpkit.MaybeValidValue
			return
		}
		panic("unreachable")
	}

	must.Assert(ctx, dbTip.Height+1 == basis.Nomination.Block_height)

	// validate chaining with previous consensus

	if basis.Nomination.Prev_block_hash != dbTip.Block_hash {
		liblog.Errorf(ctx, "cannot validate, previous block mismatch, nomination %v vs ours %v",
			libxdr.HashString(basis.Nomination.Prev_block_hash),
			libxdr.HashString(dbTip.Block_hash))
		reply <- scpkit.InvalidValue
		return
	}

	if basis.Prev_basis_hash != extTip.ConsensusBasisHash() {
		liblog.Errorf(ctx, "cannot validate, previous consensus hash mismatch, nomination %v vs ours %v",
			libxdr.HashString(basis.Prev_basis_hash),
			libxdr.HashString(extTip.ConsensusBasisHash()))
		reply <- scpkit.InvalidValue
		return
	}

	// validate timestamp

	prevStamp := protoutil.TimeFromTimepoint(extTip.ConsensusBasis().Timestamp)
	newStamp := protoutil.TimeFromTimepoint(basis.Timestamp)
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

	validatedTip, err := b.validateProposal(ctx, *dbTip, basis, value)
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
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= basis.Nomination.Block_height)
		reply <- scpkit.MaybeValidValue
		return
	}

	// cache validated blocks
	liblog.Debugf(ctx, "caching validation for block %s and consensus %s",
		libxdr.HashString(basis.Nomination.Block_hash),
		libxdr.HashString(value.Basis_hash))
	b.lk.Lock()
	b.setValidated(basis, value, validatedTip.Id)
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

	if basis.Nomination.Block_height != proto.BlockHeight(slot) {
		reply <- scpkit.InvalidValue
		return true
	}
	if xdrutil.TokenSizeOfBatch(basis.Nomination.Transactions_hash) !=
		xdrutil.TokenSizeOfBatch(basis.Nomination.Transactions_nondeterministic_data) {
		reply <- scpkit.InvalidValue
		return true
	}

	b.lk.Lock()
	dbTipOnRequest := b.tipDb
	alreadyValidated := b.getValidatedBlockID(value)
	b.lk.Unlock()

	if !protoutil.IsSaneConsensusBasis(basis) {
		liblog.Infof(ctx, "cannot validate, nomination not sane")
		reply <- scpkit.InvalidValue
		return true
	}

	if basis.Nomination.Block_height <= dbTipOnRequest.Height {
		b.validateCommittedConsensus(ctx, slot, basis, value, reply)
		return true
	}

	if alreadyValidated != nil {
		reply <- scpkit.FullyValidValue
		return true
	}

	return false
}
