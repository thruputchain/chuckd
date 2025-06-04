package nominate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
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

// PrepareNomination can be aborted by cancelling the context.
func PrepareNomination(
	ctx context.Context,
	handler batonapi.OpsHandler,
	tip batonapi.Tip,
	txs []*proto.SignedTransaction,
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

	proposedTxs, err := execPropose(ctx, handler, txs, blockID)
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

func execPropose(
	ctx context.Context,
	handler batonapi.OpsHandler,
	txs []*proto.SignedTransaction,
	blockID proto.BlockID,
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

	// buffer all proposed transactions
	txCh := make(chan *proto.SignedTransaction, len(txs))
	for _, tx := range txs {
		txCh <- tx
	}
	close(txCh)

	// create a context to use for cancelling workers in case any of them fail
	feedCtx, feedCtxCancel := context.WithCancel(ctx)
	defer feedCtxCancel()

	proposedCh := make(chan execProposed, len(execs))
	for _, exec := range execs {
		go feedExecPropose(feedCtx, exec, blockID, txCh, proposedCh)
	}

	// collect outcomes
	var proposedTxs []ProposedTx
	var verr error
	for range execs {
		pd := <-proposedCh
		if verr == nil || pd.err == batonapi.ErrNotValid { // prioritize ErrNotValid over networking errors
			verr = err
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

func feedExecPropose(
	ctx context.Context,
	exec batonapi.ExecCoordConn,
	blockID proto.BlockID,
	txCh <-chan *proto.SignedTransaction,
	proposedCh chan<- execProposed,
) {

	// return result after other defer cleanup takes place
	var result execProposed
	defer func() {
		proposedCh <- result
	}()

	// executor open proposal block
	execResult, err := exec.Executorcoord_open_proposal_block(ctx, blockID)
	if err != nil {
		result = execProposed{err: err}
		return
	}
	if execResult != proto.ExecResultOK {
		result = execProposed{err: fmt.Errorf("executor open proposal block rpc error")}
		return
	}

	// executor finish proposal block
	defer func() {
		exec.Executorcoord_finish_block(ctx, blockID)
	}()

	proposedTxs := []ProposedTx{}
	for tx := range txCh {
		select {
		case <-ctx.Done():
			result = execProposed{err: fmt.Errorf("context cancelled")}
			return
		default:
		}

		proposeResult, err := exec.Executorcoord_propose_transaction(ctx, blockID, *tx)
		if err != nil {
			proposedTxs = append(proposedTxs, ProposedTx{Tx: tx, Err: err})
		} else if proposeResult.Result != proto.ExecResultOK && proposeResult.Result != proto.ExecResultTxDeterministicError {
			proposedTxs = append(proposedTxs, ProposedTx{Tx: tx, Err: batonapi.ErrNotValid})
		} else {
			proposedTxs = append(proposedTxs, ProposedTx{Tx: tx, Nondeterministic: proposeResult.Proposal_data()})
		}
	}

	result = execProposed{err: nil, proposedTxs: proposedTxs}
	return
}
