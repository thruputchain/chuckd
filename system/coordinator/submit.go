package coordinator

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type UnvettedSubmitter struct {
	lot chan []*proto.SignedTransaction
}

func NewUnvettedSubmitter(
	ctx context.Context,
	time libtime.Time,
	maxSize int,
	maxWait time.Duration,
) *UnvettedSubmitter {
	return &UnvettedSubmitter{
		lot: make(chan []*proto.SignedTransaction),
	}
}

func (x *UnvettedSubmitter) SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction) {
	x.lot <- txs
}

func (x *UnvettedSubmitter) Start(ctx context.Context, sched proto.SchedulerCoord1) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case txs := <-x.lot:
				liblog.Debugf("submitting %d unvetted txs from wallet to scheduler", len(txs))

				var wg sync.WaitGroup
				for _, lot := range breakupTxs(txs) {
					wg.Add(1)
					go func(lot []*proto.SignedTransaction) {
						defer wg.Done()
						err := sched.Schedulercoord_submit_unvetted(ctx, protoutil.Pack(lot))
						must.NoError(ctx, err)
					}(lot)
				}
				wg.Wait()
			}
		}
	}()
}

func breakupTxs(txs []*proto.SignedTransaction) [][]*proto.SignedTransaction {
	const batchSize = 100e3 // 100K
	var lots [][]*proto.SignedTransaction
	for i := 0; i < len(txs); i += batchSize {
		end := min(i+batchSize, len(txs))
		lots = append(lots, txs[i:end])
	}
	return lots
}
