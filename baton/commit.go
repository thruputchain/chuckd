package baton

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) commit(ctx context.Context, vb *ValidatableBlock) (proto.BlockID, error) {
	if vb.AlreadyValidatedBlockID != nil {
		// skip validation when committing our own nomination
		return b.commitAlreadyValidated(ctx, vb)
	} else {
		return b.validateCommit(ctx, vb)
	}
}

func (b *Baton) commitAlreadyValidated(ctx context.Context, vb *ValidatableBlock) (proto.BlockID, error) {

	// connect to db
	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := *vb.AlreadyValidatedBlockID

	// db open snapshot
	dbResult, err := db.Dbcoord_open_snapshot(ctx, blockID)
	if err != nil {
		return 0, err
	}
	if dbResult != proto.DbResultOK {
		return 0, fmt.Errorf("db open snapshot error")
	}

	return b.finalizeCommit(ctx, vb, blockID, db)
}

func (b *Baton) finalizeCommit(
	ctx context.Context,
	vb *ValidatableBlock,
	blockID proto.BlockID,
	db DbCoordConn,
) (proto.BlockID, error) {

	// db commit snapshot
	dbBlockResult, err := db.Dbcoord_commit_snapshot(ctx, blockID)
	if err != nil {
		return 0, err
	}
	if dbBlockResult.Result != proto.DbResultOK {
		return 0, fmt.Errorf("db commit snapshot rpc error")
	}

	// verify the block hash returned by the db matches
	must.Assertf(ctx, *dbBlockResult.Block_hash() == vb.BlockHash,
		"db block hash changed after a prior proposal or validation")

	// record committed transactions to prevent replay
	for _, ptx := range vb.ProposedTransactions {
		hash := sha256.Sum256(xdrutil.MarshalXdr(&ptx.Signed_tx))
		b.handler.AddCommittedTx(ctx, hash)
		b.handler.RemoveTx(ctx, hash)
	}

	b.handler.Emit(ctx, EventCommitted{ValidatableBlock: vb, BlockID: blockID})

	return blockID, nil
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
	var props []*proto.Proposal
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
