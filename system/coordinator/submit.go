package coordinator

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Disseminator interface {
	Disseminate(ctx context.Context, submitted *proto.SubmittedTransactions)
}

type Submitter struct {
	time         libtime.Time
	pool         *mempool.SubmittedMempool
	disseminator Disseminator
	lotSize      int
	lotWait      time.Duration
	ch           chan *proto.SignedTransaction
}

func NewSubmitter(
	ctx context.Context,
	time libtime.Time,
	pool *mempool.SubmittedMempool,
	disseminator Disseminator,
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
	m := xdrutil.NewXdrMarshaler()
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

		if len(pending) > 0 {
			x.pool.AddTxs(ctx, protoutil.Unpack(pending)...)
			liblog.Debugf("disseminating %d txs: %v", len(pending),
				form.Display(protoutil.HashSignedTxs(m, pending)),
			)
			x.disseminator.Disseminate(ctx, &proto.SubmittedTransactions{Transactions: pending})
			pending = pending[:0]
		}
	}
}
