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

func (b *Baton) validateConsensus(
	ctx context.Context,
	slot scpkit.Slot,
	cv *proto.ConsensusValue,
	cvc *proto.ConsensusValueCue,
	reply chan<- scpkit.ValidationLevel,
) {

	liblog.Infof(ctx, "validating cv: height %d; block_hash %s; cv_hash %s; proposer %s",
		slot, libxdr.HashString(cv.Nomination.Block_hash), libxdr.HashString(cvc.Cv_hash),
		protoutil.VPublicShortString(cvc.Proposer))

	if b.tryValidateConsensusFast(ctx, slot, cv, cvc, reply) {
		return
	}

	// (commence condition) wait for the block below the validation request to be reached and committed

	var dbTip *proto.DbTip
	var extTip *externalizedConsensus

	var err error
	dbTip, extTip, err = b.WaitCommittedAt(ctx, cv.Nomination.Block_height-1)
	if err != nil { // context cancelled
		switch {
		case errors.Is(err, ErrTipAlreadyAhead): // from WaitCommittedAt
			must.Assert(ctx, b.HighestCommitted() >= cv.Nomination.Block_height)
			b.validateCommittedConsensus(ctx, slot, cv, cvc, reply)
			return
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded): // from WaitExternalizedBeyond
			highestExternalized := b.HighestCommittedOrExternalized()
			liblog.Infof(ctx, "skipping validation due to externalizations ahead: height: %d; highest_externalized: %d",
				cv.Nomination.Block_height, highestExternalized)
			// assert highest externalized is at cv.Nomination.Block_height or above
			must.Assert(ctx, highestExternalized >= cv.Nomination.Block_height)
			reply <- scpkit.MaybeValidValue
			return
		}
		panic("unreachable")
	}

	must.Assert(ctx, dbTip.Height+1 == cv.Nomination.Block_height)

	// validate chaining with previous consensus

	if cv.Nomination.Prev_block_hash != dbTip.Block_hash {
		liblog.Errorf(ctx, "cannot validate, previous block mismatch, nomination %v vs ours %v",
			libxdr.HashString(cv.Nomination.Prev_block_hash),
			libxdr.HashString(dbTip.Block_hash))
		reply <- scpkit.InvalidValue
		return
	}

	if cv.Prev_cv_hash != extTip.ConsensusValueHash() {
		liblog.Errorf(ctx, "cannot validate, previous consensus hash mismatch, nomination %v vs ours %v",
			libxdr.HashString(cv.Prev_cv_hash),
			libxdr.HashString(extTip.ConsensusValueHash()))
		reply <- scpkit.InvalidValue
		return
	}

	// validate timestamp

	prevStamp := protoutil.TimeFromTimepoint(extTip.ConsensusValue().Timestamp)
	newStamp := protoutil.TimeFromTimepoint(cv.Timestamp)
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
	b.lock()
	alreadyInvalidated := b.isInvalidated(cvc.Cv_hash, cvc.Block_height)
	b.unlock()
	if alreadyInvalidated {
		reply <- scpkit.InvalidValue
		return
	}

	validatedTip, err := b.validateProposal(ctx, *dbTip, cv, cvc.Cv_hash)
	if errors.Is(err, batonkit.ErrNotValid) {
		// cache invalidation to prevent dos
		b.lock()
		b.setInvalidated(cv, cvc.Cv_hash)
		b.unlock()

		reply <- scpkit.InvalidValue
		return
	}
	if err != nil {
		// err can only ever be context cancelled or deadline exceeded.
		// it occurs only if scp has externalized newer blocks
		must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= cv.Nomination.Block_height)
		reply <- scpkit.MaybeValidValue
		return
	}

	// cache validated blocks
	liblog.Debugf(ctx, "caching validation for block %s and consensus %s",
		libxdr.HashString(cv.Nomination.Block_hash),
		libxdr.HashString(cvc.Cv_hash))
	b.lock()
	b.setValidated(cv, cvc.Cv_hash, cv.Nomination.Block_height, validatedTip.Id)
	b.unlock()

	reply <- scpkit.FullyValidValue
}

// tryValidateConsensusFast returns true if it was able to make a judgement and send a reply.
func (b *Baton) tryValidateConsensusFast(
	ctx context.Context,
	slot scpkit.Slot,
	cv *proto.ConsensusValue,
	cvc *proto.ConsensusValueCue,
	reply chan<- scpkit.ValidationLevel,
) bool {

	if cv.Nomination.Block_height != proto.BlockHeight(slot) {
		reply <- scpkit.InvalidValue
		return true
	}
	if xdrutil.TokenSizeOfBatch(cv.Nomination.Transactions_hash) !=
		xdrutil.TokenSizeOfBatch(cv.Nomination.Transactions_nondeterministic_data) {
		reply <- scpkit.InvalidValue
		return true
	}

	b.lock()
	dbTipOnRequest := b.tipDb
	alreadyValidated := b.getValidatedBlockID(cvc.Cv_hash, cvc.Block_height)
	b.unlock()

	if !protoutil.IsSaneConsensusValue(cv) {
		liblog.Infof(ctx, "cannot validate, nomination not sane")
		reply <- scpkit.InvalidValue
		return true
	}

	if cv.Nomination.Block_height <= dbTipOnRequest.Height {
		b.validateCommittedConsensus(ctx, slot, cv, cvc, reply)
		return true
	}

	if alreadyValidated != nil {
		reply <- scpkit.FullyValidValue
		return true
	}

	return false
}
