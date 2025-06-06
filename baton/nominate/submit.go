package nominate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/exec"
	"github.com/groundhogchain/chuckd/util"
	proto "github.com/groundhogchain/protocol-go"
)

type ProposedTx = exec.DoneTx[*proto.SignedTransaction, *proto.NondeterministicProposalData]

type submitProposeTx struct{}

func (submitProposeTx) Buffer(txs []*proto.SignedTransaction) <-chan *proto.SignedTransaction {
	ch := make(chan *proto.SignedTransaction, len(txs))
	for _, tx := range txs {
		ch <- tx
	}
	close(ch)
	return ch
}

func (submitProposeTx) Submit(
	ctx context.Context,
	ctxCancel context.CancelFunc,
	blockID proto.BlockID,
	conn batonapi.ExecCoordConn,
	txCh <-chan *proto.SignedTransaction,
) ([]ProposedTx, error) {

	doneTxs := []ProposedTx{}
	for tx := range txCh {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}

		proposeResult, err := conn.Executorcoord_propose_transaction(ctx, blockID, *tx)
		var ptx ProposedTx
		if err != nil {
			ptx = ProposedTx{Tx: tx, Err: err}
			ctxCancel()
		} else if proposeResult.Result != proto.ExecResultOK && proposeResult.Result != proto.ExecResultTxDeterministicError {
			ptx = ProposedTx{Tx: tx, Err: batonapi.ErrNotValid}
			ctxCancel()
		} else {
			ptx = ProposedTx{Tx: tx, Result: proposeResult.Proposal_data()}
		}
		doneTxs = append(doneTxs, ptx)
	}
	return doneTxs, nil
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

func (submitProposeBatch) Submit(
	ctx context.Context,
	ctxCancel context.CancelFunc,
	blockID proto.BlockID,
	conn batonapi.ExecCoordConn,
	batchCh <-chan proto.SignedTransactionBatch,
) ([]ProposedTx, error) {

	doneTxs := []ProposedTx{}
	for batch := range batchCh {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}

		proposeResults, err := conn.Executorcoord_propose_batch(ctx, blockID, batch)

		for i, proposeResult := range proposeResults {
			var ptx ProposedTx
			if err != nil {
				ptx = ProposedTx{Tx: &batch[i], Err: err}
				ctxCancel()
			} else if proposeResult.Result != proto.ExecResultOK && proposeResult.Result != proto.ExecResultTxDeterministicError {
				ptx = ProposedTx{Tx: &batch[i], Err: batonapi.ErrNotValid}
				ctxCancel()
			} else {
				ptx = ProposedTx{Tx: &batch[i], Result: proposeResult.Proposal_data()}
			}
			doneTxs = append(doneTxs, ptx)
		}
	}

	return doneTxs, nil
}
