package baton

import (
	"context"
	"fmt"

	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) validate(ctx context.Context, vb *ValidatableBlock, db DbCoordConn, blockID proto.BlockID) error {

	// db open snapshot
	dbResult, err := db.Dbcoord_open_snapshot(ctx, vb.DbTip.ID, blockID)
	if err != nil {
		return err
	}
	if dbResult != proto.DbResultOK {
		return fmt.Errorf("db open snapshot error")
	}

	// connect to available executors
	execs, err := b.handler.DialExecs(ctx)
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return err
	}
	if len(execs) == 0 {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return ErrNoExecs
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
			execResult, err := exec.Executorcoord_open_validation_block(ctx, blockID)
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

				execResult, err := exec.Executorcoord_validate_transaction(ctx, blockID, *tx)
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
			err = exec.Executorcoord_finish_block(ctx, blockID)
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
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return fmt.Errorf("validation error: %v", verr)
	}

	return nil
}

func (b *Baton) validateCommit(ctx context.Context, vb *ValidatableBlock) error {

	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	blockID := b.handler.GenerateBlockID(ctx)

	if err = b.validate(ctx, vb, db, blockID); err != nil {
		return err
	}

	return b.finalizeCommit(ctx, vb, blockID, db)
}

func (b *Baton) validateClose(ctx context.Context, vb *ValidatableBlock) (proto.BlockID, error) {

	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := b.handler.GenerateBlockID(ctx)

	if err = b.validate(ctx, vb, db, blockID); err != nil {
		return 0, err
	}

	return blockID, b.finalizeClose(ctx, vb, blockID, db)
}

func (b *Baton) finalizeClose(
	ctx context.Context,
	vb *ValidatableBlock,
	nextID proto.BlockID,
	db DbCoordConn,
) error {

	// db close snapshot
	dbBlockResult, err := db.Dbcoord_close_snapshot(ctx, nextID)
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

	return nil
}
