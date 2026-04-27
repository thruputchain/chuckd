package coordinator

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type UnvettedSubmitter struct {
	time         libtime.Time
	maxBatchSize int
	maxWait      time.Duration
	lot          chan []*proto.SignedTransaction
	parlot       chan proto.ParallelSignedTransactionBatch
	done         chan struct{}
	once         sync.Once
}

func NewUnvettedSubmitter(
	ctx context.Context,
	time libtime.Time,
	maxBatchSize int,
	maxWait time.Duration,
) *UnvettedSubmitter {
	return &UnvettedSubmitter{
		time:         time,
		maxBatchSize: maxBatchSize,
		maxWait:      maxWait,
		lot:          make(chan []*proto.SignedTransaction),
		parlot:       make(chan proto.ParallelSignedTransactionBatch),
		done:         make(chan struct{}),
	}
}

func (x *UnvettedSubmitter) SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction) {
	select {
	case <-ctx.Done():
	case <-x.done:
	case x.lot <- txs:
	}
}

func (x *UnvettedSubmitter) SubmitParallelTxs(ctx context.Context, txs proto.ParallelSignedTransactionBatch) {
	select {
	case <-ctx.Done():
	case <-x.done:
	case x.parlot <- txs:
	}
}

func (x *UnvettedSubmitter) Stop() {
	x.once.Do(func() {
		close(x.done)
	})
}

func (x *UnvettedSubmitter) Start(ctx context.Context, sched proto.SchedulerCoord1) {
	go func() {
		m := libxdr.NewXdrMarshaler()
		buf := NewBatchingBufferForSignedTxs(
			ctx,
			x.time,
			x.maxBatchSize,
			x.maxWait,
			func(ctx context.Context, batch []proto.SignedTransaction) {
				lot := xdrutil.ParallelizeSignedTransactionBatch(
					m, systemfixed.UnvettedTxBatchMaxSegments, batch)
				err := sched.Schedulercoord_submit_unvetted(ctx, lot)
				if errors.Is(err, context.Canceled) {
					return
				}
				must.NoError(ctx, err)
			},
		)
		for {
			select {
			case <-ctx.Done():
				return
			case <-x.done:
				return
			case txs := <-x.lot:
				liblog.Debugf(ctx, "submitting %d unvetted txs from wallet to scheduler", len(txs))
				buf.Submit(ctx, protoutil.Pack(txs))
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-x.done:
				return
			case txs := <-x.parlot:
				liblog.Debugf(ctx, "submitting %d parallel unvetted txs from wallet to scheduler",
					xdrutil.TokenSizeOfBatch(txs))
				err := sched.Schedulercoord_submit_unvetted(ctx, txs)
				must.NoError(ctx, err)
			}
		}
	}()
}
