package mempool

import (
	"context"
	"sync"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

type Mempool struct {
	batchTTL time.Duration
	lk       sync.Mutex
	batch    map[proto.BatchID]*batch
}

func NewMempool(ctx context.Context, batchTTL time.Duration) *Mempool {
	mp := &Mempool{
		batchTTL: batchTTL,
		batch:    map[proto.BatchID]*batch{},
	}
	go mp.supervise(ctx)
	return mp
}

func (mp *Mempool) AddLoose(ctx context.Context, txs []proto.SignedTransaction) {
	panic("XXX")
}

func (mp *Mempool) AddBatch(ctx context.Context, batchHeader proto.BatchHeader, txs []proto.SignedTransaction) {
	panic("XXX")
}

func (mp *Mempool) LookupBatch(ctx context.Context, batchID proto.BatchID) ([]proto.SignedTransaction, bool) {
	panic("XXX")
}

func (mp *Mempool) supervise(ctx context.Context) {
	panic("XXX")
}
