package coordinator

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/jay/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type WalletSubmitter struct {
	buf *libflow.Buffer[*proto.SignedTransaction]
	lot chan []*proto.SignedTransaction
}

func NewWalletSubmitter(
	ctx context.Context,
	time libtime.Time,
	maxSize int,
	maxWait time.Duration,
) *WalletSubmitter {
	x := &WalletSubmitter{
		lot: make(chan []*proto.SignedTransaction),
	}
	x.buf = libflow.NewBuffer(ctx, time, x.flush, maxSize, maxWait)
	return x
}

func (x *WalletSubmitter) SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction) {
	x.buf.Submit(ctx, txs...)
}

func (x *WalletSubmitter) Start(ctx context.Context, sched proto.SchedulerCoord1) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case txs := <-x.lot:
				err := sched.Schedulercoord_submit_unvetted(ctx, protoutil.Pack(txs))
				must.NoError(ctx, err)
			}
		}
	}()
}

func (x *WalletSubmitter) flush(ctx context.Context, txs []*proto.SignedTransaction) {
	x.lot <- txs
}
