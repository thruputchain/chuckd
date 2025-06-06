package validate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/util"
	proto "github.com/groundhogchain/protocol-go"
)

type submitPayload[Payload any] interface {
	Buffer([]*proto.ProposedTransaction) <-chan Payload
	SubmitValidate(
		ctx context.Context,
		ctxCancel context.CancelFunc,
		blockID proto.BlockID,
		exec batonapi.ExecCoordConn,
		payCh <-chan Payload,
	) error
}

type submitValidateTx struct{}

func (submitValidateTx) Buffer(txs []*proto.ProposedTransaction) <-chan *proto.ProposedTransaction {
	ch := make(chan *proto.ProposedTransaction, len(txs))
	for _, tx := range txs {
		ch <- tx
	}
	close(ch)
	return ch
}

func (submitValidateTx) SubmitValidate(
	ctx context.Context,
	ctxCancel context.CancelFunc,
	blockID proto.BlockID,
	exec batonapi.ExecCoordConn,
	txCh <-chan *proto.ProposedTransaction,
) error {

	var result error
	for tx := range txCh {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		default:
		}

		execResult, err := exec.Executorcoord_validate_transaction(ctx, blockID, *tx)
		if execResult != proto.ExecResultOK && execResult != proto.ExecResultTxDeterministicError {
			result = batonapi.ErrNotValid
			ctxCancel()
		}
		if err != nil {
			result = err // networking error
			ctxCancel()
		}
	}
	return result
}

type submitValidateBatch struct {
	BatchSize int
}

func (x submitValidateBatch) Buffer(txs []*proto.ProposedTransaction) <-chan proto.ProposedTransactionBatch {
	ch := make(chan proto.ProposedTransactionBatch, util.CeilDivSigned(len(txs), x.BatchSize))
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
	return ch
}

func (x submitValidateBatch) SubmitValidate(
	ctx context.Context,
	ctxCancel context.CancelFunc,
	blockID proto.BlockID,
	exec batonapi.ExecCoordConn,
	batchCh <-chan proto.ProposedTransactionBatch,
) error {

	var result error
	for batch := range batchCh {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		default:
		}

		execResult, err := exec.Executorcoord_validate_batch(ctx, blockID, batch)

		for _, execResult := range execResult {
			if execResult != proto.ExecResultOK && execResult != proto.ExecResultTxDeterministicError {
				result = batonapi.ErrNotValid
				ctxCancel()
			}
		}
		if err != nil {
			result = err // networking error
			ctxCancel()
		}
	}
	return result
}
