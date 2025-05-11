package mempool

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type BatchMempool struct {
	time     libtime.Time
	batchTTL time.Duration
	lk       sync.Mutex
	batch    map[proto.BatchHash]*batch
}

func NewBatchMempoolReal(ctx context.Context, batchTTL time.Duration) *BatchMempool {
	return NewBatchMempool(ctx, libtime.RealTime{}, batchTTL)
}

func NewBatchMempool(ctx context.Context, time libtime.Time, batchTTL time.Duration) *BatchMempool {
	mp := &BatchMempool{
		time:     time,
		batchTTL: batchTTL,
		batch:    map[proto.BatchHash]*batch{},
	}
	go mp.supervise(ctx)
	return mp
}

func (mp *BatchMempool) AddBatch(ctx context.Context, txs proto.BatchTransactions) {
	hash := sha256.Sum256(xdrutil.MarshalXdr(&txs))
	mp.lk.Lock()
	defer mp.lk.Unlock()
	mp.batch[hash] = newBatch(txs.Transactions)
}

func (mp *BatchMempool) LookupBatch(ctx context.Context, batchHash proto.BatchHash) ([]*proto.ProposedTransaction, bool) {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	b, ok := mp.batch[batchHash]
	return b.txs, ok
}

func (mp *BatchMempool) supervise(_ context.Context) {
	for {
		mp.time.Sleep(mp.batchTTL)
		now := mp.time.Now()
		mp.lk.Lock()
		for k, b := range mp.batch {
			if !b.Pinned() && now.Sub(b.created) > mp.batchTTL {
				delete(mp.batch, k)
			}
		}
		mp.lk.Unlock()
	}
}
