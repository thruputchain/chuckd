package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) commit(ctx context.Context, cb *batonkit.CommittableBlock) proto.BlockID {

	validatedBlockID := cb.AlreadyValidated()

	// validate, if not already validated
	if validatedBlockID == nil {

		if txs := cb.SignedTransactions(); txs != nil {
			// submit catchup transactions
			// XXX: use scheduler rpc for network txs
			err := b.handler.Schedulercoord_submit_unvetted(ctx, txs)
			must.NoError(ctx, err)
		}

		validatedTip, err := b.validate(ctx, cb.Onto, cb.AsNomination(b.txM))
		// err can only ever be context cancelled or deadline exceeded.
		// however, this error is not expected on the commit path.
		must.NoError(ctx, err)
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
	if b.nominated != nil && b.nominated.Nomination.Block_height <= b.tip.Height {
		b.nominated = nil
	}
}
