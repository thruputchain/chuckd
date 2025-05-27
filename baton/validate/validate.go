package validate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/sdk"
	proto "github.com/groundhogchain/protocol-go"
)

type Handler interface {
	DialExecs(context.Context) ([]sdk.ExecCoordConn, error)
	Emit(ctx context.Context, event any)
}

func Validate(ctx context.Context, handler Handler, vb *sdk.ValidatableBlock, db sdk.DbCoordConn, blockID proto.BlockID) error {

	handler.Emit(ctx, sdk.EventValidating{ValidatableBlock: vb, BlockID: blockID})

	// db create snapshot
	dbResult, err := db.Dbcoord_create_snapshot(ctx, vb.DbTip.ID, blockID, proto.KBlockExecutionModeValidation)
	if err != nil {
		return err
	}
	if dbResult != proto.DbResultOK {
		return fmt.Errorf("db create snapshot error")
	}

	// connect to available executors
	execs, err := handler.DialExecs(ctx)
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return err
	}
	if len(execs) == 0 {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return sdk.ErrNoExecs
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
					validatedCh <- sdk.ErrNotValid
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
		if err := <-validatedCh; verr == nil || err == sdk.ErrNotValid { // prioritize ErrNotValid over networking errors
			verr = err
		}
	}
	if verr != nil {
		// db abandon snapshot
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return verr
	}

	handler.Emit(ctx, sdk.EventValidated{ValidatableBlock: vb, BlockID: blockID})

	return nil
}
