package system

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Submitter struct {
	time         libtime.Time
	pool         *mempool.SubmittedMempool
	disseminator batonapi.Disseminator
	lotSize      int
	lotWait      time.Duration
	ch           chan *proto.SignedTransaction
}

func NewSubmitter(
	ctx context.Context,
	time libtime.Time,
	pool *mempool.SubmittedMempool,
	disseminator batonapi.Disseminator,
	lotSize int,
	lotWait time.Duration,
) *Submitter {
	x := &Submitter{
		time:         time,
		pool:         pool,
		disseminator: disseminator,
		lotSize:      lotSize,
		lotWait:      lotWait,
		ch:           make(chan *proto.SignedTransaction),
	}
	go x.loop(ctx)
	return x
}

func (x *Submitter) SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction) {
	for _, tx := range txs {
		x.ch <- tx
	}
}

func (x *Submitter) loop(ctx context.Context) {
	for { // tx lot submission loop
		var pending []proto.SignedTransaction
		wake := x.time.After(x.lotWait)
	fill:
		for len(pending) < x.lotSize {
			select {
			case tx := <-x.ch:
				pending = append(pending, *tx)
			case <-ctx.Done():
				return
			case <-wake:
				break fill
			}
		}

		x.pool.AddTxs(ctx, protoutil.Unpack(pending)...)
		x.disseminator.Disseminate(ctx, &proto.SubmittedTransactions{Transactions: pending})
	}
}
