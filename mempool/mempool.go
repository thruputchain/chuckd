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
	batch    map[proto.BatchHash]*batch
}

func NewMempool(ctx context.Context, batchTTL time.Duration) *Mempool {
	mp := &Mempool{
		batchTTL: batchTTL,
		batch:    map[proto.BatchHash]*batch{},
	}
	go mp.supervise(ctx)
	return mp
}

// AddLoose is called by the wallet gateway to add new transaction to the mempool.
// The mempool will eventually package them into batches.
func (mp *Mempool) AddLoose(ctx context.Context, txs []proto.SignedTransaction) {
	panic("XXX")
}

func (mp *Mempool) AddBatch(ctx context.Context, batchHeader proto.BatchHeader, txs []proto.ProposedTransaction) {
	panic("XXX")
}

func (mp *Mempool) LookupBatch(ctx context.Context, batchID proto.BatchHash) ([]proto.ProposedTransaction, bool) {
	panic("XXX")
}

func (mp *Mempool) supervise(ctx context.Context) {
	panic("XXX")
}
