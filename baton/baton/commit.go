package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) commit(ctx context.Context, vb *batonkit.ValidatableBlock) proto.BlockID {

	onto := vb.Onto
	validatedBlockID := vb.AlreadyValidated

	// validate
	if validatedBlockID == nil {

		var nomination *proto.Nomination

		// if there is a catchup block,
		if vb.Catchup != nil {

			// submit catchup transactions
			err := b.handler.Schedulercoord_submit_unvetted(ctx, vb.Catchup.Signed_transactions)
			must.NoError(ctx, err)

			// create nomination block
			nomination = &proto.Nomination{
				Proposer:                b.handler.Signer(),
				Block_height:            vb.Catchup.Block_height,
				Block_hash:              vb.Catchup.Block_hash,
				Prev_block_hash:         vb.Catchup.Prev_block_hash,
				Transaction_hashes:      protoutil.HashSignedTxs(b.txM, vb.Catchup.Signed_transactions),
				Transaction_nominations: vb.Catchup.Nominated_transactions,
			}

		} else {
			nomination = &vb.Nomination
		}

		validatedTip, err := b.validate(ctx, onto, nomination)
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
		Onto:   vb.Onto,
		Height: vb.BlockHeight(),
		Hash:   vb.BlockHash(),
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
