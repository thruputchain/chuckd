package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: commit currently does not handle the upgrades in the consensus value
func (b *Baton) commit(ctx context.Context, cb *batonkit.CommittableBlock) proto.BlockID {

	liblog.Infof(ctx, "committing: height %d; block_hash %s; cv_hash %s; proposer %s",
		cb.BlockHeight(), libxdr.HashString(cb.BlockHash()), libxdr.HashString(cb.ConsensusValueHash()),
		protoutil.VPublicShortString(cb.AsConsensusValue().Nomination.Proposer))

	b.stampCommitStart(ctx, cb.BlockHeight())
	defer b.stampCommitEnd(cb.BlockHeight())

	validatedBlockID := cb.AlreadyValidated()

	// validate, if not already validated
	if validatedBlockID == nil {

		cv := cb.AsConsensusValue()
		proposer := protoutil.PublicVID(cv.Nomination.Proposer)

		if txs := cb.UnsubmittedSignedTransactions(); txs != nil {
			// submit catchup transactions
			batch := xdrutil.ParallelizeSignedTransactionBatch(b.batchCatchupM, systemfixed.CatchupTxBatchMaxSegments, txs)
			err := b.handler.Schedulercoord_submit_vetted(ctx, proposer, batch)
			must.NoError(ctx, err)
		}

		validatedTip, err := b.validateProposal(ctx, cb.Onto, cv, cb.ConsensusValueHash())
		if err != nil {
			if errors.Is(err, batonkit.ErrNotValid) {
				liblog.Fatalf(ctx, "catchup block failed validation at height %d: block hash mismatch", cb.BlockHeight())
			}
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			must.NoError(ctx, err)
		}
		validatedBlockID = &validatedTip.Id
	}

	// CV must be guaranteed persisted before the db commits.
	b.handler.CVPoolCommit(ctx, *cb.AsConsensusValue(), cb.ConsensusValueHash())

	// catchup block must be guaranteed persisted before the db commits.
	b.handler.CBPoolWaitPersist(ctx, cb.BlockHeight(), cb.ConsensusValueHash())

	// commit to db
	outcome, err := b.handler.Schedulercoord_commit_block(ctx, *validatedBlockID)
	must.NoError(ctx, err)
	must.Assertf(ctx, outcome == proto.SchedulerCommitOK, "scheduler commit outcome: %v", outcome)

	evCommitted := batonkit.EventCommitted{
		Onto:           cb.Onto,
		Height:         cb.BlockHeight(),
		BlockHash:      cb.BlockHash(),
		DbgValidatable: cb,
	}
	liblog.Infof(ctx, "%v", evCommitted)
	b.handler.Emit(ctx, evCommitted)

	return *validatedBlockID
}

func (b *Baton) purgeNotAboveTipHeight(_ context.Context) {

	// purge catchup blocks
	var catchup catchupArrivals
	for _, cb := range b.catchupArrivals {
		if cb.BlockHeight() > b.tipDb.Height {
			catchup = append(catchup, cb)
		}
	}
	b.catchupArrivals = catchup

	// purge externalized consensus values
	var exts []externalized
	for _, p := range b.externalized {
		if p.Height() > b.tipDb.Height {
			exts = append(exts, p)
		}
	}
	b.externalized = exts

	// purge validations
	for h, p := range b.validated {
		if p.Nomination.Block_height <= b.tipDb.Height {
			delete(b.validated, h)
		}
	}

	// purge invalidations
	for h, p := range b.invalidated {
		if p.Nomination.Block_height <= b.tipDb.Height {
			delete(b.invalidated, h)
		}
	}
}
