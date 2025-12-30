package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: commit currently does not handle the upgrades in the consensus value
func (b *Baton) commit(ctx context.Context, cb *batonkit.CommittableBlock) proto.BlockID {

	b.stampCommitStart(cb.BlockHeight())
	defer b.stampCommitEnd(cb.BlockHeight())

	validatedBlockID := cb.AlreadyValidated()

	// validate, if not already validated
	if validatedBlockID == nil {

		cv := cb.AsConsensusValue()
		proposer := protoutil.PublicVID(cv.Nomination.Proposer)

		if txs := cb.UnsubmittedSignedTransactions(); txs != nil {
			// submit catchup transactions
			err := b.handler.Schedulercoord_submit_vetted(ctx, proposer, txs)
			must.NoError(ctx, err)
		}

		validatedTip, err := b.validateProposal(ctx, cb.Onto, cv.Nomination)
		if err != nil {
			// err can only ever be context cancelled or deadline exceeded.
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			// however, this error is not expected on the commit path,
			// as the block being validated arises from an externalization or
			// catchup (whose hash has been confirmed by an externalization)
			// XXX: verify that the hashes in catchup blocks match the contents of the block
			must.NoError(ctx, err)
		}
		validatedBlockID = &validatedTip.Id
	}

	// commit
	// TODO: scheduler should persist catchup blocks after every commit
	// See https://github.com/groundhogchain/chuckd/issues/15
	outcome, err := b.handler.Schedulercoord_commit_block(ctx, *validatedBlockID)
	must.NoError(ctx, err)
	must.Assert(ctx, outcome == proto.SchedulerCommitOK)

	b.handler.Emit(ctx, batonkit.EventCommitted{
		Onto:   cb.Onto,
		Height: cb.BlockHeight(),
		Hash:   cb.BlockHash(),
		//
		DbgValidatable: cb,
	})

	return *validatedBlockID
}

func (b *Baton) purgeNotAboveTipHeight(ctx context.Context) {

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

	// purge partial nominations
	b.handler.UpdateNoPartialNominationsBelow(ctx, b.tipDb.Height+1)
}
