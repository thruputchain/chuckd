package validate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/commit"
	"github.com/groundhogchain/chuckd/baton/exec1"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// validation calls each executor on callsPerExec concurrent threads.
// this produces a similar effect as join-bounded-shortest-queue (JBSQ)
// in the case that there always are rpc calls ready to be fired.
// see https://www.usenix.org/conference/atc19/presentation/kogias-r2p2 JBSQ
// I would make the bound (callsPerExec) 2*N where N is the number of cores used by the executor.

func ValidateClose(
	ctx context.Context,
	handler batonapi.OpsHandler,
	vb *batonapi.ValidatableBlock,
	callsPerExec int,
	batchSize int,
) (proto.BlockID, error) {

	db, err := handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := handler.GenerateBlockID(ctx)

	if err = Validate(ctx, handler, vb, db, blockID, callsPerExec, batchSize); err != nil {
		return 0, err
	}

	return blockID, finalizeValidateClose(ctx, vb, blockID, db)
}

func ValidateCommit(
	ctx context.Context,
	txM xdrutil.XdrMarshaler,
	handler batonapi.OpsHandler,
	vb *batonapi.ValidatableBlock,
	callsPerExec int,
	batchSize int,
) (proto.BlockID, error) {
	liblog.Debugf("validating and committing [%d; %s]", vb.BlockHeight, form.Display(vb.BlockHash))

	db, err := handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := handler.GenerateBlockID(ctx)

	if err = Validate(ctx, handler, vb, db, blockID, callsPerExec, batchSize); err != nil {
		return 0, err
	}

	return commit.FinalizeCommit(ctx, txM, handler, vb, blockID, db)
}

func Validate(
	ctx context.Context,
	handler batonapi.OpsHandler,
	vb *batonapi.ValidatableBlock,
	db batonapi.DbCoordConn,
	blockID proto.BlockID,
	callsPerExec int,
	batchSize int,
) error {

	handler.Emit(ctx, batonapi.EventValidating{ValidatableBlock: vb, BlockID: blockID})

	// db create snapshot
	dbResult, err := db.Dbcoord_create_snapshot(ctx, vb.DbTip.ID, blockID, proto.KBlockExecutionModeValidation)
	if err != nil {
		return err
	}
	if dbResult != proto.DbResultOK {
		return fmt.Errorf("db create snapshot error")
	}

	if batchSize < 2 {
		err = execValidateTx(ctx, handler, vb, blockID, callsPerExec)
	} else {
		err = execValidateBatch(ctx, handler, vb, blockID, callsPerExec, batchSize)
	}
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return err
	}

	handler.Emit(ctx, batonapi.EventValidated{ValidatableBlock: vb, BlockID: blockID})

	return nil
}

func execValidateTx(
	ctx context.Context,
	handler batonapi.OpsHandler,
	vb *batonapi.ValidatableBlock,
	blockID proto.BlockID,
	callsPerExec int,
) error {
	doneTx, err := exec1.Exec(ctx, handler, vb.ProposedTransactions, blockID, submitValidateTx{}, callsPerExec)
	if err != nil {
		return err
	}
	for _, dtx := range doneTx {
		if dtx.Err != nil {
			return dtx.Err
		}
	}
	return nil
}

func execValidateBatch(
	ctx context.Context,
	handler batonapi.OpsHandler,
	vb *batonapi.ValidatableBlock,
	blockID proto.BlockID,
	callsPerExec int,
	batchSize int,
) error {
	doneTx, err := exec1.Exec(ctx, handler, vb.ProposedTransactions, blockID, submitValidateBatch{BatchSize: batchSize}, callsPerExec)
	if err != nil {
		return err
	}
	for _, dtx := range doneTx {
		if dtx.Err != nil {
			return dtx.Err
		}
	}
	return nil
}

func finalizeValidateClose(
	ctx context.Context,
	vb *batonapi.ValidatableBlock,
	blockID proto.BlockID,
	db batonapi.DbCoordConn,
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
		return batonapi.ErrNotValid
	}

	return nil
}
