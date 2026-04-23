package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
)

func (b *Baton) drive(ctx context.Context) {
	defer b.cond.Broadcast()

	v := b.buildCleanView(ctx)

	liblog.Debugf(ctx, "view: %s", v.ShortString())
	liblog.Debugf(ctx, "resource usage: %s", runtime.ResourceUsage())

	// commit next block, if one is available and no commit is in progress
	b.driveCommit(ctx, v)

	// start nomination, if there are no externalizations above the db tip and no nomination has started
	b.driveNominate(ctx, v)

	// send catchup requests
	b.driveCatchup(ctx, v)

	// send repair requests for scp envelopes if necessary
	b.driveLiveness(ctx, v)
}

// driveCommit commits the next block, if one is available and no commit is in progress
func (b *Baton) driveCommit(ctx context.Context, v *View) {
	cb := v.NextBlockToCommit()
	if cb == nil || b.committing {
		return
	}

	b.committing = true
	// cancel ongoing nominations trumped by this commit to save compute
	b.cancelNominatingNotAbove(ctx, cb.BlockHeight())

	liblog.Debugf(ctx, "COMMITTING: height %v; tx_count %d; hash %s",
		cb.BlockHeight(), cb.NumTransactions(), libxdr.HashString(cb.BlockHash()))

	go func() {
		blockID := b.commit(ctx, cb)
		{
			b.lock()

			must.Assertf(ctx, b.committing, "race committing")
			b.committing = false

			if b.tipDb.Height+1 != cb.BlockHeight() {
				panic("bug")
			}

			b.tipDb.Block_hash = cb.BlockHash()
			b.tipDb.Height = cb.BlockHeight()
			b.tipDb.Id = blockID

			b.tipExt = b.newExternalized(cb.AsConsensusValue(), cb.ConsensusValueHash())

			// purge catchup or consensus blocks no higher than the newly committed one
			b.purgeNotAboveTipHeight(ctx)

			b.noteCommitted(cb.BlockHeight())

			if b.hooks != nil {
				b.levers = b.hooks.CommitHook(cb)
			}

			b.unlock()
		}
		b.handler.Committed(ctx, cb) // must be outside of the baton lock
	}()
}

// driveNominate starts a nomination, if there are no externalizations above the db tip and no nomination has started
func (b *Baton) driveNominate(ctx context.Context, v *View) {
	if b.levers.PauseNominating {
		return
	}

	ontoDb, ontoExt := v.NominateOnto()
	if !b.canNominate(ontoDb, b.nominating) {
		return
	}

	// cancel prior finished or unfinished nominations
	if b.nominating != nil {
		b.nominating.Cancel()
		b.nominating = nil
	}
	nomCtx, nomCancel := context.WithCancel(ctx)
	current := &nominating{
		OntoDb:  *ontoDb,
		OntoExt: *ontoExt,
		Cancel: func() {
			// liblog.Debugf("cancelling ongoing nomination for slot %d", scpapi.Slot(tipToNominateOn.Height+1))
			nomCancel()
		},
	}
	b.nominating = current

	liblog.Infof(ctx, "PREPARING: height %d", ontoDb.Height+1)

	go func() {
		cv, cvc, blockID, err := b.nominate(nomCtx, *ontoDb, *ontoExt)
		current.Cancel()

		b.lock()
		defer b.unlock()

		// if nomination was superceded, b.nominating may point at a different nomination
		currentWasReplaced := b.nominating != current

		if err != nil {
			// scp rejects nominations when it is not our turn to nominate, or
			// when an externalization happens during nomination (usually only in a single-node network)
			if !errors.Is(err, ErrNominationRejected) {
				must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			}
			return
		}

		if !currentWasReplaced {
			b.setValidated(cv, cvc.Cv_hash, blockID)
			liblog.Infof(ctx, "nominated block at height %d with block hash %s",
				cv.Nomination.Block_height,
				libxdr.HashString(cv.Nomination.Block_hash),
			)
		} else {
			liblog.Infof(ctx, "superceded nomination for block at height %d with block hash %s",
				cv.Nomination.Block_height,
				libxdr.HashString(cv.Nomination.Block_hash),
			)
		}
	}()
}

// driveCatchup sends catchup requests, if needed
func (b *Baton) driveCatchup(ctx context.Context, v *View) {
	if needed := v.NeededUnrequestedSpans(); needed.IsNotEmpty() {
		liblog.Debugf(ctx, "requesting catchup: %v", needed)
		b.attemptCatchup(ctx, needed)
	}
}

func (b *Baton) driveLiveness(ctx context.Context, v *View) {
	now := b.handler.Now()
	sinceExt := b.sinceLastExternalization(now)
	sinceBackfeed := b.sinceLastBackfeed(now)
	if sinceExt < b.patience || sinceBackfeed < b.patience {
		return
	}

	slot := scpkit.Slot(v.HighestExternalizedOrCommitted() + 1)
	liblog.Infof(ctx, "requesting backfeed for slot %d", slot)
	b.noteBackfeed()
	b.handler.BackfeedSCPEnvelopes(ctx, slot, systemfixed.SCPBackfeedTimeout)
}
