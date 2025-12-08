package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtext"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) validateConsensus(
	ctx context.Context,
	slot scpkit.Slot,
	cv *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) {

	liblog.Infof("validating consensus value at height %d with block hash %s",
		slot, libtext.Display(cv.Nomination.Block_hash))

	if b.tryValidateConsensusFast(ctx, slot, cv, reply) {
		return
	}

	// (commence condition) wait for the block below the validation request to be reached and committed

	var dbTip *proto.DbTip
	var extTip *externalizedConsensus

	var err error
	dbTip, extTip, err = b.WaitCommittedAt(ctx, cv.Nomination.Block_height-1)
	if err != nil { // context cancelled
		must.Assert(ctx,
			errors.Is(err, ErrTipAlreadyAhead) || // from WaitCommittedAt
				errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded), // from WaitExternalizedBeyond
		)
		b.validateExternalizedConsensus(ctx, slot, cv, reply)
		return
	}

	must.Assert(ctx, dbTip.Height+1 == cv.Nomination.Block_height)

	// validate chaining with previous consensus

	if cv.Nomination.Prev_block_hash != dbTip.Hash {
		liblog.Infof("cannot validate, previous block mismatch, nomination %v vs ours %v",
			libtext.Display(cv.Nomination.Prev_block_hash),
			libtext.Display(dbTip.Hash))
		reply <- scpkit.InvalidValue
		return
	}

	if cv.Prev_cv_hash != extTip.ConsensusValueHash() {
		liblog.Infof("cannot validate, previous consensus hash mismatch, nomination %v vs ours %v",
			libtext.Display(cv.Prev_cv_hash),
			libtext.Display(extTip.ConsensusValueHash()))
		reply <- scpkit.InvalidValue
		return
	}

	// validate timestamp

	prevStamp := protoutil.TimeFromTimepoint(extTip.ConsensusValue().Timestamp)
	newStamp := protoutil.TimeFromTimepoint(cv.Timestamp)
	if !prevStamp.Before(newStamp) {
		liblog.Infof("cannot validate, timestamp not increasing")
		reply <- scpkit.InvalidValue
		return
	}
	now := b.handler.Now()
	if !newStamp.Before(now) {
		liblog.Infof("cannot validate, timestamp in the future (stamp %s, now %s)",
			libtime.FmtTime(newStamp), libtime.FmtTime(now))
		reply <- scpkit.InvalidValue
		return
	}

	// validate proposal

	// check if block previous invalidated to prevent expensive validation
	b.lock()
	alreadyInvalidated := b.isInvalidatedCV(cv)
	b.unlock()
	if alreadyInvalidated {
		reply <- scpkit.InvalidValue
		return
	}

	validatedTip, err := b.validateProposal(ctx, *dbTip, cv.Nomination)
	if errors.Is(err, batonkit.ErrNotValid) {
		// cache invalidation to prevent dos
		b.lock()
		b.setInvalidated(cv)
		b.unlock()

		reply <- scpkit.InvalidValue
		return
	}
	if err != nil {
		// err can only ever be context cancelled or deadline exceeded.
		must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		// it occurs only if scp has externalized newer blocks
		b.validateExternalizedConsensus(ctx, slot, cv, reply)
		return
	}

	// cache validated blocks
	liblog.Debugf("caching validation for block %s and consensus %s",
		libtext.Display(cv.Nomination.Block_hash),
		libtext.Display(b.hashCV(cv)))
	b.lock()
	b.setValidated(cv, validatedTip.Id)
	b.unlock()

	reply <- scpkit.FullyValidValue
}

// tryValidateConsensusFast returns true if it was able to make a judgement and send a reply.
func (b *Baton) tryValidateConsensusFast(
	ctx context.Context,
	slot scpkit.Slot,
	cv *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) bool {

	if cv.Nomination.Block_height != proto.BlockHeight(slot) {
		reply <- scpkit.InvalidValue
		return true
	}
	if len(cv.Nomination.Transaction_hashes) != len(cv.Nomination.Transaction_nominations) {
		reply <- scpkit.InvalidValue
		return true
	}

	b.lock()
	dbTipOnRequest := b.tipDb
	alreadyValidated := b.getValidatedBlockIDFromCV(cv)
	b.unlock()

	if !protoutil.IsSaneConsensusValue(cv) {
		liblog.Infof("cannot validate, nomination not sane")
		reply <- scpkit.InvalidValue
		return true
	}

	if cv.Nomination.Block_height <= dbTipOnRequest.Height {
		b.validateExternalizedConsensus(ctx, slot, cv, reply)
		return true
	}

	if alreadyValidated != nil {
		reply <- scpkit.FullyValidValue
		return true
	}

	return false
}
