package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) commit(ctx context.Context, cb *batonkit.CommittableBlock) proto.BlockID {

	validatedBlockID := cb.AlreadyValidated()

	// validate, if not already validated
	if validatedBlockID == nil {

		nomination := cb.AsNomination(b.txM)
		proposer := protoutil.PublicVID(nomination.Proposer)

		if txs := cb.SignedTransactions(); txs != nil {
			// submit catchup transactions
			err := b.handler.Schedulercoord_submit_vetted(ctx, proposer, txs)
			must.NoError(ctx, err)
		}

		validatedTip, err := b.validate(ctx, cb.Onto, nomination)
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
	err := b.handler.Schedulercoord_commit_block(ctx, *validatedBlockID)
	must.NoError(ctx, err)

	b.handler.Emit(ctx, batonkit.EventCommitted{
		Onto:   cb.Onto,
		Height: cb.BlockHeight(),
		Hash:   cb.BlockHash(),
		//
		DbgValidatable: cb,
	})

	return *validatedBlockID
}

func (b *Baton) purgeNotAboveTipHeight(_ context.Context) {

	// purge catchup blocks
	var catchup catchupBlocks
	for _, cb := range b.catchupBlocks {
		if cb.BlockHeight() > b.tip.Height {
			catchup = append(catchup, cb)
		}
	}
	b.catchupBlocks = catchup

	// purge consensus blocks
	var props []*proto.Nomination
	for _, p := range b.externalized {
		if p.Block_height > b.tip.Height {
			props = append(props, p)
		}
	}
	b.externalized = props

	// purge validations
	for h, p := range b.validated {
		if p.Block_height <= b.tip.Height {
			delete(b.validated, h)
		}
	}

	// purge nominations
	if b.nominated != nil && b.nominated.ConsensusValue.Nomination.Block_height <= b.tip.Height {
		b.nominated = nil
	}
}
