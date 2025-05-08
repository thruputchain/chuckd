package mempool

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/util/xdrutil"
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

func (mp *ProposedMempool) AddBatch(ctx context.Context, txs proto.BatchTransactions) {
	hash := sha256.Sum256(xdrutil.MarshalXdr(&txs))
	mp.lk.Lock()
	defer mp.lk.Unlock()
	mp.batch[hash] = newBatch(txs.Transactions)
}

func (mp *ProposedMempool) LookupBatch(ctx context.Context, batchHash proto.BatchHash) ([]*proto.ProposedTransaction, bool) {
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
