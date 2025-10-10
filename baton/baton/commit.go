package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) commit(ctx context.Context, vb *batonapi.ValidatableBlock) proto.BlockID {

	onto := vb.Onto.Proto()
	validatedBlockID := vb.AlreadyValidated

	// validate
	if validatedBlockID == nil {
		// XXX: if there is a catchup block, submit its txs to mempool first
		XXX

		validatedTip, err := b.validate(ctx, onto, &vb.Nomination)
		must.NoError(ctx, err)
		validatedBlockID = &validatedTip.Id
	}

	// commit
	err := b.handler.Schedulercoord_commit_block(ctx, *validatedBlockID)
	must.NoError(ctx, err)

	// record catchup block
	b.handler.Committed(ctx, &proto.CatchupBlock{
		Block_height:          vb.BlockHeight(),
		Block_hash:            vb.BlockHash(),
		Prev_block_hash:       vb.PrevBlockHash(),
		Proposed_transactions: protoutil.PackProposedTransactions(vb.ProposedTransactions),
	})

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
