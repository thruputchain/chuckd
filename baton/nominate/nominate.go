package nominate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/util"
	proto "github.com/groundhogchain/protocol-go"
)

type Nomination struct {
	BlockID       proto.BlockID
	BlockHeight   proto.BlockHeight
	BlockHash     proto.BlockHash
	PrevBlockHash proto.BlockHash
	ProposedTxs   []ProposedTx
}

func (x Nomination) ProposedTransactions() []proto.ProposedTransaction {
	ptxs := []proto.ProposedTransaction{}
	for _, ptx := range x.ProposedTxs {
		if ptx.Err != nil {
			continue
		}
		ptxs = append(ptxs, proto.ProposedTransaction{
			Signed_tx:                      *ptx.Tx,
			Nondeterministic_proposal_data: *ptx.Nondeterministic,
		})
	}
	return ptxs
}

type ProposedTx struct {
	Tx               *proto.SignedTransaction
	Nondeterministic *proto.NondeterministicProposalData
	Err              error
}

type execProposed struct {
	err         error
	proposedTxs []ProposedTx
}

// nomination calls each executor on callsPerExec concurrent threads.
// this produces a similar effect as join-bounded-shortest-queue (JBSQ)
// in the case that there always are rpc calls ready to be fired.
// see https://www.usenix.org/conference/atc19/presentation/kogias-r2p2 JBSQ
// I would make the bound (callsPerExec) 2*N where N is the number of cores used by the executor.

// PrepareNomination can be aborted by cancelling the context.
func PrepareNomination(
	ctx context.Context,
	handler batonapi.OpsHandler,
	tip batonapi.Tip,
	txs []*proto.SignedTransaction,
	callsPerExec int,
	batchSize int,
) (*Nomination, error) {

	// connect to db
	db, err := handler.DialDb(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	blockID := handler.GenerateBlockID(ctx)

	// db create snapshot
	dbResult, err := db.Dbcoord_create_snapshot(ctx, tip.ID, blockID, proto.KBlockExecutionModeProposal)
	if err != nil {
		return nil, err
	}
	if dbResult != proto.DbResultOK {
		return nil, fmt.Errorf("db create snapshot rpc error")
	}

	var proposedTxs []ProposedTx
	if batchSize < 2 {
		proposedTxs, err = execProposeTx(ctx, handler, txs, blockID, callsPerExec)
	} else {
		proposedTxs, err = execProposeBatch(ctx, handler, txs, blockID, callsPerExec, batchSize)
	}
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return nil, fmt.Errorf("proposal error: %v", err)
	}

	// db close snapshot
	dbBlockResult, err := db.Dbcoord_close_snapshot(ctx, blockID)
	if err != nil {
		return nil, err
	}
	if dbBlockResult.Result != proto.DbResultOK {
		return nil, fmt.Errorf("db close snapshot rpc error")
	}

	return &Nomination{
		BlockID:       blockID,
		BlockHeight:   tip.Height + 1,
		BlockHash:     *dbBlockResult.Block_hash(),
		PrevBlockHash: tip.Hash,
		ProposedTxs:   proposedTxs,
	}, nil
}

func execProposeTx(
	ctx context.Context,
	handler batonapi.OpsHandler,
	txs []*proto.SignedTransaction,
	blockID proto.BlockID,
	callsPerExec int,
) ([]ProposedTx, error) {
	return execPropose(
		ctx,
		handler,
		txs,
		blockID,
		submitProposeTx{},
		callsPerExec,
	)
}

func execProposeBatch(
	ctx context.Context,
	handler batonapi.OpsHandler,
	txs []*proto.SignedTransaction,
	blockID proto.BlockID,
	callsPerExec int,
	batchSize int,
) ([]ProposedTx, error) {
	return execPropose(
		ctx,
		handler,
		txs,
		blockID,
		submitProposeBatch{BatchSize: batchSize},
		callsPerExec,
	)
}

func execPropose[Payload any, Submit submitPayload[Payload]](
	ctx context.Context,
	handler batonapi.OpsHandler,
	txs []*proto.SignedTransaction,
	blockID proto.BlockID,
	submit Submit,
	callsPerExec int,
) ([]ProposedTx, error) {

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

	return doExecPropose(ctx, blockID, txs, execs, submit, callsPerExec)
}

func doExecPropose[Payload any, Submit submitPayload[Payload]](
	ctx context.Context,
	blockID proto.BlockID,
	txs []*proto.SignedTransaction,
	execs []batonapi.ExecCoordConn,
	submit Submit,
	callsPerExec int,
) ([]ProposedTx, error) {

	// create a context to use for cancelling workers in case any of them fail
	feedCtx, feedCtxCancel := context.WithCancel(ctx)
	defer feedCtxCancel()

	proposedCh := make(chan execProposed, len(execs)*callsPerExec)

	payCh := submit.Buffer(txs)

	for _, exec := range execs {
		for i := 0; i < callsPerExec; i++ {
			go feedExecPropose[Payload, Submit](feedCtx, exec, blockID, payCh, proposedCh)
		}
	}

	return collectExecPropose(proposedCh, len(execs)*callsPerExec, feedCtxCancel)
}

func collectExecPropose(
	proposedCh <-chan execProposed,
	numResponses int,
	feedCtxCancel context.CancelFunc,
) ([]ProposedTx, error) {

	var proposedTxs []ProposedTx
	var verr error
	for i := 0; i < numResponses; i++ {
		pd := <-proposedCh
		if verr == nil || pd.err == batonapi.ErrNotValid { // prioritize ErrNotValid over networking errors
			verr = pd.err
		}
		if pd.err == nil {
			proposedTxs = append(proposedTxs, pd.proposedTxs...)
		}
		if verr != nil {
			feedCtxCancel() // cancel workers context to stop the other feeders if any of them fail
		}
	}

	if verr != nil {
		return nil, verr
	}

	return proposedTxs, nil
}

func feedExecPropose[Payload any, Submit submitPayload[Payload]](
	ctx context.Context,
	exec batonapi.ExecCoordConn,
	blockID proto.BlockID,
	payCh <-chan Payload,
	proposedCh chan<- execProposed,
) {

	execResult, err := exec.Executorcoord_open_proposal_block(ctx, blockID)
	if err != nil {
		proposedCh <- execProposed{err: err}
		return
	}
	if execResult != proto.ExecResultOK {
		proposedCh <- execProposed{err: fmt.Errorf("executor open proposal block rpc error")}
		return
	}

	var s Submit
	proposedTxs, err := s.SubmitPropose(ctx, blockID, exec, payCh)

	exec.Executorcoord_finish_block(ctx, blockID)

	proposedCh <- execProposed{err: err, proposedTxs: proposedTxs}
}

//

type submitPayload[Payload any] interface {
	Buffer([]*proto.SignedTransaction) <-chan Payload
	SubmitPropose( //XXX: must be able to cancel context from inside
		ctx context.Context,
		blockID proto.BlockID,
		exec batonapi.ExecCoordConn,
		payCh <-chan Payload,
	) ([]ProposedTx, error)
}

type submitProposeTx struct{}

func (submitProposeTx) Buffer(txs []*proto.SignedTransaction) <-chan *proto.SignedTransaction {
	ch := make(chan *proto.SignedTransaction, len(txs))
	for _, tx := range txs {
		ch <- tx
	}
	close(ch)
	return ch
}

func (submitProposeTx) SubmitPropose(
	ctx context.Context,
	blockID proto.BlockID,
	exec batonapi.ExecCoordConn,
	txCh <-chan *proto.SignedTransaction,
) ([]ProposedTx, error) {

	proposedTxs := []ProposedTx{}
	for tx := range txCh {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}

		proposeResult, err := exec.Executorcoord_propose_transaction(ctx, blockID, *tx)
		var ptx ProposedTx
		if err != nil {
			ptx = ProposedTx{Tx: tx, Err: err}
		} else if proposeResult.Result != proto.ExecResultOK && proposeResult.Result != proto.ExecResultTxDeterministicError {
			ptx = ProposedTx{Tx: tx, Err: batonapi.ErrNotValid}
		} else {
			ptx = ProposedTx{Tx: tx, Nondeterministic: proposeResult.Proposal_data()}
		}
		proposedTxs = append(proposedTxs, ptx)
	}
	return proposedTxs, nil
}

type submitProposeBatch struct {
	BatchSize int
}

func (x submitProposeBatch) Buffer(txs []*proto.SignedTransaction) <-chan proto.SignedTransactionBatch {
	ch := make(chan proto.SignedTransactionBatch, util.CeilDivSigned(len(txs), x.BatchSize))
	batch := make(proto.SignedTransactionBatch, 0, x.BatchSize)
	for _, tx := range txs {
		if len(batch) == x.BatchSize {
			ch <- batch
			batch = make(proto.SignedTransactionBatch, 0, x.BatchSize)
		}
		batch = append(batch, *tx)
	}
	if len(batch) > 0 {
		ch <- batch
	}
	close(ch)
	return ch
}

func (submitProposeBatch) SubmitPropose(
	ctx context.Context,
	blockID proto.BlockID,
	exec batonapi.ExecCoordConn,
	batchCh <-chan proto.SignedTransactionBatch,
) ([]ProposedTx, error) {

	proposedTxs := []ProposedTx{}
	for batch := range batchCh {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}

		proposeResults, err := exec.Executorcoord_propose_batch(ctx, blockID, batch)

		for i, proposeResult := range proposeResults {
			var ptx ProposedTx
			if err != nil {
				ptx = ProposedTx{Tx: &batch[i], Err: err}
			} else if proposeResult.Result != proto.ExecResultOK && proposeResult.Result != proto.ExecResultTxDeterministicError {
				ptx = ProposedTx{Tx: &batch[i], Err: batonapi.ErrNotValid}
			} else {
				ptx = ProposedTx{Tx: &batch[i], Nondeterministic: proposeResult.Proposal_data()}
			}
			proposedTxs = append(proposedTxs, ptx)
		}
	}

	return proposedTxs, nil
}
