package baton

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type committed struct{}

func (b *Baton) startValidateAndCommit(ctx context.Context) {
	for {
		if err := b.doCommit(ctx, <-b.commit); err != nil {
			log.Errorf("validate and commit: %v", err)
		}
	}
}

func (b *Baton) doCommit(ctx context.Context, vb *ValidatableBlock) error {
	if vb.AlreadyValidatedBlockID != nil {
		// skip validation when committing our own nomination
		return b.commitAlreadyValidated(ctx, vb)
	} else {
		return b.validateCommit(ctx, vb)
	}
}

func (b *Baton) commitAlreadyValidated(ctx context.Context, vb *ValidatableBlock) error {

	// connect to db
	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	blockID := *vb.AlreadyValidatedBlockID

	// db open snapshot
	dbResult, err := db.Dbcoord_open_snapshot(ctx, blockID)
	if err != nil {
		return err
	}
	if dbResult != proto.DbResultOK {
		return fmt.Errorf("db open snapshot error")
	}

	return b.finalizeCommit(ctx, vb, blockID, db)
}

func (b *Baton) finalizeCommit(
	ctx context.Context,
	vb *ValidatableBlock,
	blockID proto.BlockID,
	db DbCoordConn,
) error {

	// db commit snapshot
	dbBlockResult, err := db.Dbcoord_commit_snapshot(ctx, blockID)
	if err != nil {
		return err
	}
	if dbBlockResult.Result != proto.DbResultOK {
		return fmt.Errorf("db commit snapshot rpc error")
	}

	// verify the block hash returned by the db matches
	if *dbBlockResult.Block_hash() != vb.BlockHash {
		return fmt.Errorf("block hash validation mismatch")
	}

	// record committed transactions to prevent replay
	for _, ptx := range vb.ProposedTransactions {
		hash := sha256.Sum256(xdrutil.MarshalXdr(&ptx.Signed_tx))
		b.handler.AddCommittedTx(ctx, hash)
		b.handler.RemoveTx(ctx, hash)
	}

	// notify baton
	defer func() { b.committed <- committed{} }()

	// update state
	b.lock()
	defer b.unlock()

	if b.db.tip.Height+1 != vb.BlockHeight {
		panic("bug")
	}

	// only this function writes to the state, and sequentially
	b.db.tip.Hash = vb.BlockHash
	b.db.tip.Height = vb.BlockHeight
	b.db.tip.ID = blockID

	// purge catchup or consensus blocks no higher than the newly committed one
	b.purgeBelowTip(ctx)

	b.handler.Emit(ctx, EventCommitted{ValidatableBlock: vb, BlockID: blockID})

	return nil
}

func (b *Baton) purgeBelowTip(_ context.Context) {

	// purge catchup blocks
	var catchup catchupBlocks
	for _, cb := range b.catchup.blocks {
		if cb.BlockHeight() > b.db.tip.Height {
			catchup = append(catchup, cb)
		}
	}
	b.catchup.blocks = catchup

	// purge consensus blocks
	var props proposals
	for _, p := range b.consensus.ext {
		if p.BlockHeight() > b.db.tip.Height {
			props = append(props, p)
		}
	}
	b.consensus.ext = props

	// purge validations
	for h, p := range b.consensus.validatedProposal {
		if p.Block_height <= b.db.tip.Height {
			delete(b.consensus.validatedProposal, h)
			delete(b.consensus.validatedBlockID, h)
		}
	}

	// purge nomination blocks
	if b.nominator.nominatedProposal != nil && b.nominator.nominatedProposal.Block_height <= b.db.tip.Height {
		b.nominator.nominatedProposal = nil
		b.nominator.nominatedBlockID = nil
	}
}
