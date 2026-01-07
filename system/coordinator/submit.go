package coordinator

import (
	"context"
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
	time    libtime.Time
	maxSize int
	maxWait time.Duration
	lot     chan []*proto.SignedTransaction
}

func NewUnvettedSubmitter(
	ctx context.Context,
	time libtime.Time,
	maxSize int,
	maxWait time.Duration,
) *UnvettedSubmitter {
	return &UnvettedSubmitter{
		time:    time,
		maxSize: maxSize,
		maxWait: maxWait,
		lot:     make(chan []*proto.SignedTransaction),
	}
}

func (x *UnvettedSubmitter) SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction) {
	x.lot <- txs
}

func (x *UnvettedSubmitter) Start(ctx context.Context, sched proto.SchedulerCoord1) {
	go func() {
		m := libxdr.NewXdrMarshaler()
		buf := NewBatchingBufferForSignedTxs(
			ctx,
			x.time,
			x.maxSize,
			x.maxWait,
			func(ctx context.Context, batch []proto.SignedTransaction) {
				lot := xdrutil.ParallelizeSignedTransactionBatch(
					m, systemfixed.UnvettedTxBatchMaxSegments, batch)
				err := sched.Schedulercoord_submit_unvetted(ctx, lot)
				must.NoError(ctx, err)
			},
		)
		for {
			select {
			case <-ctx.Done():
				return
			case txs := <-x.lot:
				liblog.Debugf("submitting %d unvetted txs from wallet to scheduler", len(txs))
				buf.Submit(ctx, protoutil.Pack(txs))
			}
		}
	}()
}
