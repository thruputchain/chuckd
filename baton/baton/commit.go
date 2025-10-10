package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) commit(ctx context.Context, vb *batonapi.ValidatableBlock) proto.BlockID {

	onto := vb.Onto.Proto()
	validatedBlockID := vb.AlreadyValidated

	// validate
	if validatedBlockID == nil {
		// if there is a catchup block, validate using dedicated scheduler method
		if vb.Catchup != nil {
			panic("XXX: not implemented")
		} else {
			validatedTip, err := b.validate(ctx, onto, &vb.Nomination)
			must.NoError(ctx, err)
			validatedBlockID = &validatedTip.Id
		}
	}

	// commit
	// XXX: scheduler should persist catchup blocks after every commit
	err := b.handler.Schedulercoord_commit_block(ctx, *validatedBlockID)
	must.NoError(ctx, err)

	b.handler.Emit(ctx, batonapi.EventCommitted{ValidatableBlock: vb, BlockID: *validatedBlockID})

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
