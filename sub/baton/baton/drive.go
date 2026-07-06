package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) drive(ctx context.Context) {
	defer b.cond.Broadcast()

	v := b.buildCleanView(ctx)

	liblog.Debugf(ctx, "view: %s", v.ShortString())
	liblog.Debugf(ctx, "resource usage: %s", runtime.ResourceUsage())

	b.driveCommit(ctx, v)
	b.driveNominate(ctx, v)
	b.driveCatchup(ctx, v)
	b.driveLiveness(ctx, v)
}

func (b *Baton) driveCommit(ctx context.Context, v *View) {
	cb := v.NextBlockToCommit()
	if cb == nil || b.committing != nil {
		return
	}

	committingHeight := cb.BlockHeight()
	b.committing = &committingHeight
	b.cancelNominatingNotAbove(ctx, cb.BlockHeight())

	liblog.Debugf(ctx, "COMMITTING: height %v; tx_count %d; value %s",
		cb.BlockHeight(), cb.NumTransactions(), protoutil.ValueDisplay(cb.ConsensusValue()))

	go func() {
		blockID := b.commit(ctx, cb)
		{
			b.lock()

			must.Assertf(ctx, b.committing != nil, "race committing")
			b.committing = nil

			if protoutil.TipHeight(b.tipDb)+1 != cb.BlockHeight() {
				panic("bug")
			}

			b.tipDb = proto.DbTip{Id: blockID, Value: *cb.ConsensusValue()}
			b.tipExt = newExternalized(cb.ConsensusValue(), cb.AsConsensusBasis())
			b.purgeNotAboveTipHeight(ctx)
			b.noteCommitted(cb.BlockHeight())

			if b.hooks != nil {
				b.levers = b.hooks.CommitHook(cb)
			}

			b.unlock()
		}
		b.handler.Committed(ctx, cb)
	}()
}

func (b *Baton) driveNominate(ctx context.Context, v *View) {
	if b.levers.PauseNominating {
		return
	}

	ontoDb, ontoExt := v.NominateOnto()
	if !b.canNominate(ontoDb, b.nominating) {
		return
	}

	if b.nominating != nil {
		b.nominating.Cancel()
		b.nominating = nil
	}
	nomCtx, nomCancel := context.WithCancel(ctx)
	current := &nominating{
		OntoDb:  *ontoDb,
		OntoExt: *ontoExt,
		Cancel:  nomCancel,
	}
	b.nominating = current

	liblog.Infof(ctx, "PREPARING: height %d", protoutil.TipHeight(*ontoDb)+1)

	go func() {
		basis, value, blockID, err := b.nominate(nomCtx, *ontoDb, *ontoExt)
		current.Cancel()

		b.lock()
		defer b.unlock()

		currentWasReplaced := b.nominating != current

		if err != nil {
			if !errors.Is(err, ErrNominationRejected) {
				must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			}
			return
		}

		if !currentWasReplaced {
			b.setValidated(basis, value, blockID)
			liblog.Infof(ctx, "nominated block at height %d with value %s",
				basis.Value.Proposal_header.Block_height,
				protoutil.ValueDisplay(value),
			)
		} else {
			liblog.Infof(ctx, "superceded nomination for block at height %d with value %s",
				basis.Value.Proposal_header.Block_height,
				protoutil.ValueDisplay(value),
			)
		}
	}()
}

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
