package baton

import (
	"context"
	"errors"
	"time"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
)

func (b *Baton) drive(ctx context.Context) {
	defer b.cond.Broadcast()

	v := b.buildCleanView(ctx)

	liblog.Debugf("baton view: %s", v.ShortString())
	// liblog.Debugf("resource usage: %s", runtime.ResourceUsage())

	// commit next block, if one is available and no commit is in progress
	if cb := v.NextBlockToCommit(); cb != nil && !b.committing {
		liblog.Debugf("baton committing [height %v; hash %s]", cb.BlockHeight(), form.Display(cb.BlockHash()))
		b.committing = true
		// cancel ongoing nominations trumped by this commit to save compute
		b.cancelNominatingNotAbove(ctx, cb.BlockHeight())
		go func() {
			blockID := b.commit(ctx, cb)

			b.lock()
			defer b.unlock()

			must.Assertf(ctx, b.committing, "race committing")
			b.committing = false

			if b.tip.Height+1 != cb.BlockHeight() {
				panic("bug")
			}

			b.tip.Hash = cb.BlockHash()
			b.tip.Height = cb.BlockHeight()
			b.tip.Id = blockID

			// purge catchup or consensus blocks no higher than the newly committed one
			b.purgeNotAboveTipHeight(ctx)
		}()
	}

	// start nomination, if there are no externalizations above the db tip and no nomination has started
	if tipToNominateOn := v.TipToNominateOn(); b.canNominate(tipToNominateOn, b.nominating, b.nominated) {
		// liblog.Debugf("baton nominating onto %v", tipToNominateOn)
		// cancel prior unfinished nominations
		if b.nominating != nil {
			b.nominating.Cancel()
			b.nominating = nil
		}
		nomCtx, nomCancel := context.WithCancel(ctx)
		current := &nominating{
			Onto: *tipToNominateOn,
			Cancel: func() {
				// liblog.Debugf("baton cancelling ongoing nomination for slot %d", scpapi.Slot(tipToNominateOn.Height+1))
				nomCancel()
			},
		}
		b.nominating = current
		go func() {
			cv, blockID, err := b.nominate(nomCtx, *tipToNominateOn)
			current.Cancel()

			b.lock()
			defer b.unlock()

			// if nomination was cancelled, b.nominating may point at a different nomination
			currentWasReplaced := b.nominating != current
			if !currentWasReplaced {
				b.nominating = nil
			}

			if err != nil {
				must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
				return
			}

			if !currentWasReplaced {
				b.nominated = &nominated{
					Time:           time.Unix(0, int64(cv.Timestamp)),
					ConsensusValue: cv,
				}
				b.setValidated(cv, blockID)
				liblog.Infof("nominated block at height %d with hash %s",
					cv.Nomination.Block_height, form.Display(cv.Nomination.Block_hash))
			}
		}()
	}

	// send catchup requests
	if needed := v.NeededUnrequestedSpans(); needed.IsNotEmpty() {
		liblog.Debugf("baton requesting catchup")
		b.requestCatchup(ctx, needed)
	}
}
