package baton

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	proto "github.com/groundhogchain/protocol-go"
)

type validated struct {
	*proto.Proposal
	proto.BlockID
}

var ErrNotValid = fmt.Errorf("proposal not valid")

func (b *Baton) validate(ctx context.Context, vb *sdk.ValidatableBlock, db sdk.DbCoordConn, blockID proto.BlockID) error {

	b.handler.Emit(ctx, EventValidating{ValidatableBlock: vb, BlockID: blockID})

	// db create snapshot
	dbResult, err := db.Dbcoord_create_snapshot(ctx, vb.DbTip.ID, blockID, proto.KBlockExecutionModeValidation)
	if err != nil {
		return err
	}
	if dbResult != proto.DbResultOK {
		return fmt.Errorf("db create snapshot error")
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
		go func(exec sdk.ExecCoordConn) {

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
					validatedCh <- err // networking error
					return
				}
				if execResult != proto.ExecResultOK {
					validatedCh <- ErrNotValid
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
		if err := <-validatedCh; verr == nil || err == ErrNotValid { // prioritize ErrNotValid over networking errors
			verr = err
		}
	}
	if verr != nil {
		// db abandon snapshot
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return verr
	}

	b.handler.Emit(ctx, EventValidated{ValidatableBlock: vb, BlockID: blockID})

	return nil
}

func (b *Baton) validateCommit(ctx context.Context, vb *sdk.ValidatableBlock) (proto.BlockID, error) {
	liblog.Debugf("validating and committing [%d; %s]", vb.BlockHeight, form.Display(vb.BlockHash))

	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := b.handler.GenerateBlockID(ctx)

	if err = b.validate(ctx, vb, db, blockID); err != nil {
		return 0, err
	}

	return b.finalizeCommit(ctx, vb, blockID, db)
}

func (b *Baton) validateClose(ctx context.Context, vb *sdk.ValidatableBlock) (proto.BlockID, error) {

	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := b.handler.GenerateBlockID(ctx)

	if err = b.validate(ctx, vb, db, blockID); err != nil {
		return 0, err
	}

	return blockID, b.finalizeValidateClose(ctx, vb, blockID, db)
}

func (b *Baton) finalizeValidateClose(
	ctx context.Context,
	vb *sdk.ValidatableBlock,
	blockID proto.BlockID,
	db sdk.DbCoordConn,
) error {

	// db close snapshot
	dbBlockResult, err := db.Dbcoord_close_snapshot(ctx, blockID)
	if err != nil {
		return err
	}
	if dbBlockResult.Result != proto.DbResultOK {
		return fmt.Errorf("db commit snapshot rpc error")
	}

	// verify the block hash returned by the db matches
	if *dbBlockResult.Block_hash() != vb.BlockHash {
		liblog.Infof("cannot validate, resulting block hash %v != proposed block hash %v",
			form.Display(*dbBlockResult.Block_hash()), form.Display(vb.BlockHash))
		return ErrNotValid
	}

	return nil
}
