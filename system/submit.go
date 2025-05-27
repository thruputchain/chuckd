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
	var pending []*proto.SignedTransaction
	for { // bag loop
		wake := x.time.After(x.bagWait)
		for len(pending) < x.bagSize {
			select {
			case tx := <-x.ch:
				XXX
			case <-ctx.Done():
				XXX
			case <-wake:
				XXX
			}
		}
	}
}
