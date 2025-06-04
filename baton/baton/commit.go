package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/commit"
	"github.com/groundhogchain/chuckd/baton/validate"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) commit(ctx context.Context, vb *batonapi.ValidatableBlock) (proto.BlockID, error) {
	if vb.AlreadyValidatedBlockID != nil {
		// skip validation when committing our own nomination
		return commit.CommitAlreadyValidated(ctx, b.handler, vb)
	} else {
		return validate.ValidateCommit(ctx, b.handler, vb, b.callsPerExec)
	}
}

func (b *Baton) purgeBelowTip(_ context.Context) {

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
	if b.nominated != nil && b.nominated.Block_height <= b.tip.Height {
		b.nominated = nil
	}
}
