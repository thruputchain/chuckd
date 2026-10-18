package coordinator

import (
	"context"
	"time"

	"github.com/thruputchain/chuckd/util/xdrutil"
	"github.com/thruputchain/lib-go/libflow"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libtime"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func NewBatchingBufferForParallelSignedTxs(
	ctx context.Context,
	time libtime.Time,
	maxBatchSize int,
	maxWait time.Duration,
	flush func(context.Context, proto.ParallelSignedTransactionBatch),
) *libflow.BatchingBuffer[proto.ParallelSignedTransactionBatch] {
	buf := libflow.NewBatchingBuffer(
		ctx,
		time,
		func(ctx context.Context, batches []proto.ParallelSignedTransactionBatch) {
			liblog.Debugf(ctx, "disseminating %d vetted txs", xdrutil.TokenSizeOfBatches(batches))
			flush(ctx, xdrutil.JoinBatches(batches))
		},
		maxBatchSize,
		maxWait,
		func(batch proto.ParallelSignedTransactionBatch) int {
			return xdrutil.TokenSizeOfBatch(batch)
		},
	)
	return buf
}

func NewBatchingBufferForSignedTxs(
	ctx context.Context,
	time libtime.Time,
	maxBatchSize int,
	maxWait time.Duration,
	flush func(context.Context, []proto.SignedTransaction),
) *libflow.BatchingBuffer[[]proto.SignedTransaction] {
	buf := libflow.NewBatchingBuffer(
		ctx,
		time,
		func(ctx context.Context, batches [][]proto.SignedTransaction) {
			n := 0
			for _, b := range batches {
				n += len(b)
			}
			join := make([]proto.SignedTransaction, 0, n)
			for _, b := range batches {
				join = append(join, b...)
			}
			liblog.Debugf(ctx, "submitting %d unvetted txs from wallet to scheduler", len(join))
			flush(ctx, join)
		},
		maxBatchSize,
		maxWait,
		func(batch []proto.SignedTransaction) int {
			return len(batch)
		},
	)
	return buf
}
