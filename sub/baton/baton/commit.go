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

	value := cb.ConsensusValue()
	liblog.Infof(ctx, "committing: height %d; block_hash %s; value %s; proposer %s",
		cb.BlockHeight(), libxdr.HashString(cb.BlockHash()),
		protoutil.ValueDisplay(value),
		protoutil.VIDString(cb.AsConsensusBasis().Proposer))

	b.stampCommitStart(ctx, cb.BlockHeight())
	defer b.stampCommitEnd(cb.BlockHeight())

	validatedBlockID := cb.AlreadyValidated()

	// validate, if not already validated
	if validatedBlockID == nil {

		basis := cb.AsConsensusBasis()
		proposer := basis.Proposer

		if txs := cb.UnsubmittedSignedTransactions(); txs != nil {
			// submit catchup transactions
			batch := xdrutil.ParallelizeSignedTransactionBatch(b.batchCatchupM, systemfixed.CatchupTxBatchMaxSegments, txs)
			err := b.handler.Schedulercoord_submit_vetted(ctx, proposer, batch)
			must.NoError(ctx, err)
		}

		validatedTip, err := b.validateProposal(ctx, cb.Onto, basis, cb.ConsensusValue())
		// an error cannot arise here:
		// - context cancellation or deadlines do not apply on the commit path
		// - ErrNotValid is not expected on the commit path,
		// as the block being validated arises from an externalization or
		// catchup (whose hash has been linked to a following externalization)
		if errors.Is(err, batonkit.ErrNotValid) {
			// if the catchup block is verified (which it is), this should never happen
			// unless a quorum of validators maliciously externalized an invalid block.
			liblog.Fatalf(ctx, "committing catchup failed to validate: %v", err)
		}
		must.NoError(ctx, err)
		validatedBlockID = &validatedTip.Id
	}

	// CV must be guaranteed persisted before the db commits.
	b.handler.CVPoolCommit(ctx, *cb.AsConsensusBasis(), *cb.ConsensusValue())

	// catchup block must be guaranteed persisted before the db commits.
	b.handler.CBPoolWaitPersist(ctx, *cb.ConsensusValue())

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
