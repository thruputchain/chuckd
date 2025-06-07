package exec1

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	proto "github.com/groundhogchain/protocol-go"

	"github.com/panjf2000/ants/v2"
)

type Executor[Tx any, Result any, Pkg any, Submit SubmitPkg[Tx, Result, Pkg]] struct {
	pool *ants.Pool
}

type DoneTx[Tx any, Result any] struct {
	Tx     Tx
	Result Result
	Err    error
}

type executed[Tx any, Result any] struct {
	err         error
	executedTxs []DoneTx[Tx, Result]
}

func Exec[Tx any, Result any, Pkg any, Submit SubmitPkg[Tx, Result, Pkg]](
	ctx context.Context,
	handler batonapi.OpsHandler,
	txs []Tx,
	blockID proto.BlockID,
	submit Submit,
	callsPerExec int,
) ([]DoneTx[Tx, Result], error) {

	// connect to available executors
	execs, err := handler.DialExecs(ctx)
	if err != nil {
		return nil, err
	}
	if len(execs) == 0 {
		return nil, batonapi.ErrNoExecs
	}
	for _, exec := range execs {
		defer exec.Close()
	}

	return execWithConns(ctx, blockID, txs, execs, submit, callsPerExec)
}

func execWithConns[Tx any, Result any, Pkg any, Submit SubmitPkg[Tx, Result, Pkg]](
	ctx context.Context,
	blockID proto.BlockID,
	txs []Tx,
	conn []batonapi.ExecCoordConn,
	submit Submit,
	callsPerExec int,
) ([]DoneTx[Tx, Result], error) {

	// create a context to use for cancelling workers in case any of them fail
	feedCtx, feedCtxCancel := context.WithCancel(ctx)
	defer feedCtxCancel()

	executedCh := make(chan executed[Tx, Result], len(conn)*callsPerExec)

	payCh := submit.Buffer(txs)

	for _, conn := range conn {
		for i := 0; i < callsPerExec; i++ {
			go feedConn[Tx, Result, Pkg, Submit](feedCtx, feedCtxCancel, conn, blockID, payCh, executedCh)
		}
	}

	return collectResults(executedCh, len(conn)*callsPerExec)
}

func collectResults[Tx any, Result any](
	executedCh <-chan executed[Tx, Result],
	numResponses int,
) ([]DoneTx[Tx, Result], error) {

	var executedTxs []DoneTx[Tx, Result]
	var verr error
	for i := 0; i < numResponses; i++ {
		xd := <-executedCh
		if verr == nil || xd.err == batonapi.ErrNotValid { // prioritize ErrNotValid over networking errors
			verr = xd.err
		}
		if xd.err == nil {
			executedTxs = append(executedTxs, xd.executedTxs...)
		}
	}

	if verr != nil {
		return nil, verr
	}

	return executedTxs, nil
}

func feedConn[Tx any, Result any, Pkg any, Submit SubmitPkg[Tx, Result, Pkg]](
	ctx context.Context,
	ctxCancel context.CancelFunc,
	exec batonapi.ExecCoordConn,
	blockID proto.BlockID,
	payCh <-chan Pkg,
	executedCh chan<- executed[Tx, Result],
) {

	execResult, err := exec.Executorcoord_open_proposal_block(ctx, blockID)
	if err != nil {
		executedCh <- executed[Tx, Result]{err: err}
		return
	}
	if execResult != proto.ExecResultOK {
		executedCh <- executed[Tx, Result]{err: fmt.Errorf("executor open proposal block rpc error")}
		return
	}

	var s Submit
	executedTxs, err := s.Submit(ctx, ctxCancel, blockID, exec, payCh)

	exec.Executorcoord_finish_block(ctx, blockID)

	executedCh <- executed[Tx, Result]{err: err, executedTxs: executedTxs}
}
