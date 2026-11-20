package baton

import (
	"context"
	"errors"

	"github.com/thruputchain/chuckd/runtime"
	"github.com/thruputchain/chuckd/sub/scp/scpkit"
	"github.com/thruputchain/chuckd/system/systemfixed"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
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
	if cb == nil || b.committing != nil {
		return
	}

	committingHeight := cb.BlockHeight()
	b.committing = &committingHeight
	// cancel ongoing nominations trumped by this commit to save compute
	b.cancelNominatingNotAbove(ctx, cb.BlockHeight())

	liblog.Debugf(ctx, "COMMITTING: height %v; tx_count %d; value %s",
		cb.BlockHeight(), cb.NumTransactions(), libxdr.HashString(cb.ValueHash()))

	go func() {
		blockID := b.commit(ctx, cb)
		{
			b.lock()

			must.Assertf(ctx, b.committing != nil, "race committing")
			b.committing = nil

			if b.tipHeight()+1 != cb.BlockHeight() {
				panic("bug")
			}

			b.tipDb.Id = blockID
			b.tipDb.Value = *cb.ConsensusValue()
			b.metrics.recordCommittedBlock(ctx, cb.BlockHeight())

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

	ontoDb := v.ProposeOnto()
	if !b.canPropose(ontoDb, b.nominating) {
		return
	}

	// cancel prior finished or unfinished nominations
	if b.nominating != nil {
		b.nominating.Cancel()
		b.nominating = nil
	}
	nomCtx, nomCancel := context.WithCancel(ctx)
	current := &proposing{
		OntoDb: *ontoDb,
		Cancel: func() {
			// liblog.Debugf("cancelling ongoing nomination for slot %d", scpapi.Slot(tipToNominateOn.Height+1))
			nomCancel()
		},
	}
	b.nominating = current

	liblog.Debugf(ctx, "PREPARING: height %d", protoutil.ValueHeight(&ontoDb.Value)+1)

	go func() {
		basis, value, blockID, err := b.propose(nomCtx, *ontoDb)
		current.Cancel()

		b.lock()
		defer b.unlock()

		// if nomination was superceded, b.nominating may point at a different nomination
		currentWasReplaced := b.nominating != current

		if err != nil {
			// scp rejects nominations when it is not our turn to nominate, or
			// when an externalization happens during nomination (usually only in a single-node network)
			if !errors.Is(err, ErrProposalRejected) {
				must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			}
			return
		}

		if !currentWasReplaced {
			b.setValidated(basis, value, blockID)
			liblog.Debugf(ctx, "nominated block at height %d with value hash %s",
				protoutil.ValueHeight(&basis.Value),
				protoutil.ValueDisplay(value),
			)
		} else {
			liblog.Debugf(ctx, "superceded nomination for block at height %d with value hash %s",
				protoutil.ValueHeight(&basis.Value),
				protoutil.ValueDisplay(value),
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
	b.handler.SCPBackfeedEnvelopes(ctx, slot, systemfixed.SCPBackfeedTimeout)
}
