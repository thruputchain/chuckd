package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtext"
	"github.com/groundhogchain/lib-go/must"
)

func (b *Baton) drive(ctx context.Context) {
	defer b.cond.Broadcast()

	v := b.buildCleanView(ctx)

	liblog.Debugf("view: %s", v.ShortString())
	liblog.Debugf("resource usage: %s", runtime.ResourceUsage())

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
	if cb := v.NextBlockToCommit(); cb != nil && !b.committing {
		liblog.Debugf("committing [height %v; hash %s]", cb.BlockHeight(), libtext.Display(cb.BlockHash()))
		b.committing = true
		// cancel ongoing nominations trumped by this commit to save compute
		b.cancelNominatingNotAbove(ctx, cb.BlockHeight())

		// log duration between nomination complete to commit begins
		if b.nominated != nil && b.nominated.BlockHeight() == cb.BlockHeight() {
			liblog.Infof("duration between completion of nomination and start of commit for height %d is %v",
				cb.BlockHeight(), b.handler.Now().Sub(b.nominated.Time))
		}

		go func() {
			blockID := b.commit(ctx, cb)

			b.lock()
			defer b.unlock()

			must.Assertf(ctx, b.committing, "race committing")
			b.committing = false

			if b.tipDb.Height+1 != cb.BlockHeight() {
				panic("bug")
			}

			b.tipDb.Hash = cb.BlockHash()
			b.tipDb.Height = cb.BlockHeight()
			b.tipDb.Id = blockID

			b.tipExt = b.newExternalized(cb.AsConsensusValue())
			b.noteCommitted(cb.BlockHeight())

			// purge catchup or consensus blocks no higher than the newly committed one
			b.purgeNotAboveTipHeight(ctx)
		}()
	}
}

// driveNominate starts a nomination, if there are no externalizations above the db tip and no nomination has started
func (b *Baton) driveNominate(ctx context.Context, v *View) {
	if ontoDb, ontoExt := v.NominateOnto(); b.canNominate(ontoDb, b.nominating, b.nominated) {
		// liblog.Debugf("nominating onto %v", tipToNominateOn)
		// cancel prior unfinished nominations
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
		go func() {
			cv, blockID, err := b.nominate(nomCtx, *ontoDb, *ontoExt)
			current.Cancel()

			b.lock()
			defer b.unlock()

			// if nomination was cancelled, b.nominating may point at a different nomination
			currentWasReplaced := b.nominating != current
			if !currentWasReplaced {
				b.nominating = nil
			}

			if err != nil {
				// scp rejects nominations when it is not our turn to nominate
				if errors.Is(err, ErrNominationRejected) {
					if !currentWasReplaced {
						height := ontoDb.Height + 1
						b.nominated = &nominated{
							Time:     b.handler.Now(),
							Rejected: &height,
						}
					}
				} else {
					must.Assert(ctx, errors.Is(err, sched.ErrHeightTooLow) ||
						errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
				}
				return
			}

			if !currentWasReplaced {
				b.nominated = &nominated{
					Time:           protoutil.TimeFromTimepoint(cv.Timestamp),
					ConsensusValue: cv,
				}
				b.setValidated(cv, blockID)
				liblog.Infof("nominated block at height %d with block hash %s and consensus value hash %s",
					cv.Nomination.Block_height,
					libtext.Display(cv.Nomination.Block_hash),
					libtext.Display(b.hashCV(cv)),
				)
			}
		}()
	}
}

// driveCatchup sends catchup requests, if needed
func (b *Baton) driveCatchup(ctx context.Context, v *View) {
	if needed := v.NeededUnrequestedSpans(); needed.IsNotEmpty() {
		liblog.Debugf("requesting catchup: %v", needed)
		b.requestCatchup(ctx, needed)
	}
}

func (b *Baton) driveLiveness(ctx context.Context, v *View) {
	now := b.handler.Now()
	sinceExt := b.sinceLastExternalization(now)
	sinceBackfeed := b.sinceLastBackfeed(now)
	if sinceExt < b.patience || sinceBackfeed < b.patience {
		return
	}

	liblog.Debugf("requesting backfeed")
	b.noteBackfeed()
	b.handler.BackfeedSCPEnvelopes(ctx, scpkit.Slot(v.HighestExternalizedOrCommitted()+1))
}
