package coordinator

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type WalletSubmitter struct {
	lot chan []*proto.SignedTransaction
}

func NewWalletSubmitter(
	ctx context.Context,
	time libtime.Time,
	maxSize int,
	maxWait time.Duration,
) *WalletSubmitter {
	return &WalletSubmitter{
		lot: make(chan []*proto.SignedTransaction),
	}
}

func (x *WalletSubmitter) SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction) {
	x.lot <- txs
}

func (x *WalletSubmitter) Start(ctx context.Context, sched proto.SchedulerCoord1) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case txs := <-x.lot:
				liblog.Debugf("submitting %d unvetted txs from wallet to scheduler", len(txs))
				err := sched.Schedulercoord_submit_unvetted(ctx, protoutil.Pack(txs))
				must.NoError(ctx, err)
			}
		}
	}()
}
