package validate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/exec1"
	"github.com/groundhogchain/chuckd/util"
	proto "github.com/groundhogchain/protocol-go"
)

type ValidatedTx = exec1.DoneTx[*proto.ProposedTransaction, struct{}]

type submitValidateTx struct{}

func (submitValidateTx) Buffer(txs []*proto.ProposedTransaction) <-chan *proto.ProposedTransaction {
	ch := make(chan *proto.ProposedTransaction, len(txs))
	go func() {
		for _, tx := range txs {
			ch <- tx
		}
		close(ch)
	}()
	return ch
}

func (submitValidateTx) Submit(
	ctx context.Context,
	ctxCancel context.CancelFunc,
	blockID proto.BlockID,
	conn batonapi.ExecCoordConn,
	txCh <-chan *proto.ProposedTransaction,
) ([]ValidatedTx, error) {

	doneTxs := []ValidatedTx{}
	for tx := range txCh {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}

		validateResult, err := conn.Executorcoord_validate_transaction(ctx, blockID, *tx)
		var vtx ValidatedTx
		if err != nil {
			vtx = ValidatedTx{Tx: tx, Err: err}
			ctxCancel()
		} else if validateResult != proto.ExecResultOK && validateResult != proto.ExecResultTxDeterministicError {
			vtx = ValidatedTx{Tx: tx, Err: batonapi.ErrNotValid}
			ctxCancel()
		} else {
			vtx = ValidatedTx{Tx: tx}
		}
		doneTxs = append(doneTxs, vtx)
	}
	return doneTxs, nil
}

type submitValidateBatch struct {
	BatchSize int
}

func (x submitValidateBatch) Buffer(txs []*proto.ProposedTransaction) <-chan proto.ProposedTransactionBatch {
	ch := make(chan proto.ProposedTransactionBatch, util.CeilDivSigned(len(txs), x.BatchSize))
	go func() {
		batch := make(proto.ProposedTransactionBatch, 0, x.BatchSize)
		for _, tx := range txs {
			if len(batch) == x.BatchSize {
				ch <- batch
				batch = make(proto.ProposedTransactionBatch, 0, x.BatchSize)
			}
			batch = append(batch, *tx)
		}
		if len(batch) > 0 {
			ch <- batch
		}
		close(ch)
	}()
	return ch
}

func (x submitValidateBatch) Submit(
	ctx context.Context,
	ctxCancel context.CancelFunc,
	blockID proto.BlockID,
	conn batonapi.ExecCoordConn,
	batchCh <-chan proto.ProposedTransactionBatch,
) ([]ValidatedTx, error) {

	doneTxs := []ValidatedTx{}
	for batch := range batchCh {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}

		validateResults, err := conn.Executorcoord_validate_batch(ctx, blockID, batch)

		for i, proposeResult := range validateResults {
			var ptx ValidatedTx
			if err != nil {
				ptx = ValidatedTx{Tx: &batch[i], Err: err}
				ctxCancel()
			} else if proposeResult != proto.ExecResultOK && proposeResult != proto.ExecResultTxDeterministicError {
				ptx = ValidatedTx{Tx: &batch[i], Err: batonapi.ErrNotValid}
				ctxCancel()
			} else {
				ptx = ValidatedTx{Tx: &batch[i]}
			}
			doneTxs = append(doneTxs, ptx)
		}
	}

	return doneTxs, nil
}
