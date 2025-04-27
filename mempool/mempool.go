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

func (mp *Mempool) Add(ctx context.Context, batchHeader proto.BatchHeader, txs []proto.Transaction) {
	panic("XXX")
}

func (mp *Mempool) Lookup(ctx context.Context, batchID proto.BatchID) ([]proto.Transaction, bool) {
	panic("XXX")
}

func (mp *Mempool) supervise(ctx context.Context) {
	panic("XXX")
}
