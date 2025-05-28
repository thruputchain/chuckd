package system

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	proto "github.com/groundhogchain/protocol-go"
)

type Submitter struct {
	time         libtime.Time
	poolbag      *mempool.BagMempool
	disseminator sdk.BagDisseminator
	bagSize      int
	bagWait      time.Duration
	ch           chan *proto.SignedTransaction
}

func NewSubmitter(
	ctx context.Context,
	time libtime.Time,
	poolbag *mempool.BagMempool,
	disseminator sdk.BagDisseminator,
	bagSize int,
	bagWait time.Duration,
) *Submitter {
	x := &Submitter{
		time:         time,
		poolbag:      poolbag,
		disseminator: disseminator,
		bagSize:      bagSize,
		bagWait:      bagWait,
		ch:           make(chan *proto.SignedTransaction),
	}
	go x.loop(ctx)
	return x
}

func (x *Submitter) SubmitTx(ctx context.Context, tx *proto.SignedTransaction) {
	x.ch <- tx
}

func (x *Submitter) loop(ctx context.Context) {
	for { // bag loop
		var pending []proto.SignedTransaction
		wake := x.time.After(x.bagWait)
	fill:
		for len(pending) < x.bagSize {
			select {
			case tx := <-x.ch:
				pending = append(pending, *tx)
			case <-ctx.Done():
				return
			case <-wake:
				break fill
			}
		}

		bag := proto.BagTransactions{Transactions: pending}
		x.poolbag.AddBag(ctx, bag)
		x.disseminator.DisseminateBag(ctx, &proto.BagMessage{Bag_transactions: bag})
	}
}
