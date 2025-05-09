package baton

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type committed struct{}

type ValidatableBlock struct {
	FromCatchup   bool
	FromConsensus bool

	BlockHeight          proto.BlockHeight
	BlockHash            proto.BlockHash
	PrevBlockHash        proto.BlockHash
	ProposedTransactions []*proto.ProposedTransaction

	DbTipHeight proto.BlockHeight
	DbTipHash   proto.BlockHash
	DbTipID     proto.BlockID

	NominationProposal *proto.Proposal
	NominationBlockID  *proto.BlockID
}

func (b *Baton) startValidateAndCommit(ctx context.Context) {
	for {
		if err := b.doCommit(ctx, <-b.commit); err != nil {
			log.Errorf("validate and commit: %v", err)
		}
	}
}

func (b *Baton) doCommit(ctx context.Context, vb *ValidatableBlock) error {
	if vb.NominationBlockID != nil {
		// skip validation when committing our own nomination
		return b.commitNomination(ctx, vb)
	} else {
		return b.validateAndCommit(ctx, vb)
	}
}

func (b *Baton) commitNomination(ctx context.Context, vb *ValidatableBlock) error {

	// connect to db
	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	nextID := *vb.NominationBlockID

	// db open snapshot
	dbResult, err := db.Dbcoord_open_snapshot(ctx, vb.DbTipID, nextID)
	if err != nil {
		return err
	}
	if dbResult != proto.DbResultOK {
		return fmt.Errorf("db open snapshot error")
	}

	return b.finalizeCommit(ctx, vb, nextID, db)
}

func (b *Baton) validateAndCommit(ctx context.Context, vb *ValidatableBlock) error {

	// connect to db
	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	nextID := b.handler.GenerateBlockID(ctx)

	// db open snapshot
	dbResult, err := db.Dbcoord_open_snapshot(ctx, vb.DbTipID, nextID)
	if err != nil {
		return err
	}
	if dbResult != proto.DbResultOK {
		return fmt.Errorf("db open snapshot error")
	}

	// connect to available executors
	execs, err := b.handler.DialExecs(ctx)
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, nextID)
		return err
	}
	for _, exec := range execs {
		defer exec.Close()
	}

	// validate all transactions
	txCh := make(chan *proto.ProposedTransaction, len(vb.ProposedTransactions))
	for _, tx := range vb.ProposedTransactions {
		txCh <- tx
	}
	close(txCh)

	validatedCh := make(chan error, len(execs))
	for _, exec := range execs {
		go func(exec ExecCoordConn) {

			// executor open validation block
			execResult, err := exec.Executorcoord_open_validation_block(ctx, nextID)
			if err != nil {
				validatedCh <- err
				return
			}
			if execResult != proto.ExecResultOK {
				validatedCh <- fmt.Errorf("executor open validation block rpc error")
				return
			}

			for tx := range txCh {
				select {
				case <-ctx.Done():
					validatedCh <- fmt.Errorf("context cancelled")
				default:
				}

				execResult, err := exec.Executorcoord_validate_transaction(ctx, nextID, *tx)
				if err != nil {
					validatedCh <- err
					return
				}
				if execResult != proto.ExecResultOK {
					validatedCh <- fmt.Errorf("executor validate transaction rpc error")
					return
				}
			}

			// executor finish validation block
			err = exec.Executorcoord_finish_block(ctx, nextID)
			if err != nil {
				validatedCh <- fmt.Errorf("executor finish error: %v", err)
				return
			}

			validatedCh <- nil

		}(exec)
	}
	var verr error
	for range execs {
		if err := <-validatedCh; err != nil {
			verr = err
		}
	}
	if verr != nil {
		// db abandon snapshot
		db.Dbcoord_abandon_snapshot(ctx, nextID)
		return fmt.Errorf("validation error: %v", verr)
	}

	return b.finalizeCommit(ctx, vb, nextID, db)
}

func (b *Baton) finalizeCommit(
	ctx context.Context,
	vb *ValidatableBlock,
	nextID proto.BlockID,
	db DbCoordConn,
) error {

	// db commit snapshot
	dbBlockResult, err := db.Dbcoord_commit_snapshot(ctx, nextID)
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

	if b.db.tipHeight+1 != vb.BlockHeight {
		panic("bug")
	}

	// only this function writes to the state, and sequentially
	b.db.tipHash = vb.BlockHash
	b.db.tipHeight = vb.BlockHeight
	b.db.tipID = nextID

	// purge catchup or consensus blocks no higher than the newly committed one
	b.purgeBelowTip(ctx)

	return nil
}

func (b *Baton) purgeBelowTip(_ context.Context) {

	// purge catchup blocks
	var catchup catchupBlocks
	for _, cb := range b.catchup.blocks {
		if cb.BlockHeight() > b.db.tipHeight {
			catchup = append(catchup, cb)
		}
	}
	b.catchup.blocks = catchup

	// purge consensus blocks
	var props proposals
	for _, p := range b.consensus.prop {
		if p.BlockHeight() > b.db.tipHeight {
			props = append(props, p)
		}
	}
	b.consensus.prop = props

	// purge nomination blocks
	if b.nominator.latestProposal != nil && b.nominator.latestProposal.Block_height <= b.db.tipHeight {
		b.nominator.latestProposal = nil
		b.nominator.latestBlockID = nil
	}
}

func (b *Baton) invalidProposalExternalized(ctx context.Context, vb *ValidatableBlock) {
	if vb.FromCatchup {
		b.purgeCatchup(ctx)
	}
	if vb.FromConsensus {
		must.Panic(ctx, fmt.Errorf("invalid proposal externalized"))
	}
}
