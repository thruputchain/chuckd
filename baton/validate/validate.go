package validate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/commit"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
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

	return commit.FinalizeCommit(ctx, handler, vb, blockID, db)
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
	return execValidate(ctx, handler, vb, blockID, submitValidateTx{}, callsPerExec)
}

func execValidateBatch(
	ctx context.Context,
	handler batonapi.OpsHandler,
	vb *batonapi.ValidatableBlock,
	blockID proto.BlockID,
	callsPerExec int,
	batchSize int,
) error {
	return execValidate(ctx, handler, vb, blockID, submitValidateBatch{BatchSize: batchSize}, callsPerExec)
}

func execValidate[Payload any, Submit submitPayload[Payload]](
	ctx context.Context,
	handler batonapi.OpsHandler,
	vb *batonapi.ValidatableBlock,
	blockID proto.BlockID,
	submit Submit,
	callsPerExec int,
) error {

	// connect to available executors
	execs, err := handler.DialExecs(ctx)
	if err != nil {
		return err
	}
	if len(execs) == 0 {
		return batonapi.ErrNoExecs
	}
	for _, exec := range execs {
		defer exec.Close()
	}

	return doExecValidateTx(ctx, vb, blockID, execs, submit, callsPerExec)
}

func doExecValidateTx[Payload any, Submit submitPayload[Payload]](
	ctx context.Context,
	vb *batonapi.ValidatableBlock,
	blockID proto.BlockID,
	execs []batonapi.ExecCoordConn,
	submit Submit,
	callsPerExec int,
) error {

	// create a context to use for cancelling workers in case any of them fail
	feedCtx, feedCtxCancel := context.WithCancel(ctx)
	defer feedCtxCancel()

	validatedCh := make(chan error, len(execs)*callsPerExec)

	payCh := submit.Buffer(vb.ProposedTransactions)

	for _, exec := range execs {
		for i := 0; i < callsPerExec; i++ {
			go feedExecValidate[Payload, Submit](feedCtx, exec, blockID, payCh, validatedCh)
		}
	}

	return collectExecValidate(validatedCh, len(execs)*callsPerExec, feedCtxCancel)
}

func collectExecValidate(
	validatedCh <-chan error,
	numResponses int,
	feedCtxCancel context.CancelFunc,
) error {

	// collect outcomes
	var verr error
	for i := 0; i < numResponses; i++ {
		err := <-validatedCh
		if verr == nil || err == batonapi.ErrNotValid { // prioritize ErrNotValid over networking errors
			verr = err
		}
		if verr != nil {
			feedCtxCancel() // cancel workers context to stop the other feeders if any of them fail
		}
	}

	return verr
}

func feedExecValidate[Payload any, Submit submitPayload[Payload]](
	ctx context.Context,
	exec batonapi.ExecCoordConn,
	blockID proto.BlockID,
	payCh <-chan Payload,
	validatedCh chan<- error,
) {

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

	var s Submit
	err = s.SubmitValidate(ctx, blockID, exec, payCh)

	exec.Executorcoord_finish_block(ctx, blockID)

	validatedCh <- err
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
