package mempool

import (
	"context"
	"sync"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

type ProposedMempool struct {
	batchTTL time.Duration
	lk       sync.Mutex
	batch    map[proto.BatchHash]*batch
}

func NewProposedMempool(ctx context.Context, batchTTL time.Duration) *ProposedMempool {
	mp := &ProposedMempool{
		batchTTL: batchTTL,
		batch:    map[proto.BatchHash]*batch{},
	}
	go mp.supervise(ctx)
	return mp
}

func (mp *ProposedMempool) AddBatch(ctx context.Context, batchHeader proto.BatchHeader, txs []proto.ProposedTransaction) {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	mp.batch[batchHeader.Batch_hash] = newBatch(batchHeader, txs)
}

func (mp *ProposedMempool) LookupBatch(ctx context.Context, batchHash proto.BatchHash) ([]proto.ProposedTransaction, bool) {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	b, ok := mp.batch[batchHash]
	return b.txs, ok
}

func (mp *ProposedMempool) supervise(_ context.Context) {
	for {
		time.Sleep(mp.batchTTL)
		now := time.Now()
		mp.lk.Lock()
		for k, b := range mp.batch {
			if !b.Pinned() && now.Sub(b.created) > mp.batchTTL {
				delete(mp.batch, k)
			}
		}
		mp.lk.Unlock()
	}
}
