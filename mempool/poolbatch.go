package mempool

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type BatchMempool struct {
	time     libtime.Time
	batchTTL time.Duration
	arriving chan arrivingBatch
	waiting  chan *waitingBatch
	lk       sync.Mutex
	batch    map[proto.BatchHash]*batch
}

type arrivingBatch struct {
	hash proto.BatchHash
	ptxs []*proto.ProposedTransaction
}

type waitingBatch struct {
	ctx   context.Context
	hash  proto.BatchHash
	reply chan<- []*proto.ProposedTransaction
}

func NewBatchMempoolReal(ctx context.Context, batchTTL time.Duration) *BatchMempool {
	return NewBatchMempool(ctx, libtime.RealTime{}, batchTTL)
}

func NewBatchMempool(ctx context.Context, time libtime.Time, batchTTL time.Duration) *BatchMempool {
	mp := &BatchMempool{
		time:     time,
		batchTTL: batchTTL,
		arriving: make(chan arrivingBatch),
		waiting:  make(chan *waitingBatch),
		batch:    map[proto.BatchHash]*batch{},
	}
	go mp.supervise(ctx)
	go mp.wait(ctx)
	return mp
}

func (mp *BatchMempool) BatchCount() int {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	return len(mp.batch)
}

func (mp *BatchMempool) AddBatch(ctx context.Context, txs proto.BatchTransactions) {
	hash := sha256.Sum256(xdrutil.MarshalXdr(&txs))
	mp.lk.Lock()
	b := newBatch(mp.time.Now(), txs.Transactions)
	mp.batch[hash] = b
	mp.lk.Unlock()
	mp.arriving <- arrivingBatch{hash: hash, ptxs: b.txs}
}

func (mp *BatchMempool) LookupBatch(ctx context.Context, batchHash proto.BatchHash) ([]*proto.ProposedTransaction, bool) {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	b, ok := mp.batch[batchHash]
	if ok {
		return b.txs, true
	} else {
		return nil, false
	}
}

func (mp *BatchMempool) wait(ctx context.Context) {

	hashWaiters := map[proto.BatchHash]map[*waitingBatch]bool{}
	defer func() {
		for _, waiters := range hashWaiters {
			for waiter := range waiters {
				close(waiter.reply)
			}
		}
	}()

	for {
		// cleanup cancelled waiters
		for h, waiters := range hashWaiters {
			for waiter := range waiters {
				select {
				case <-waiter.ctx.Done():
					delete(waiters, waiter)
				default:
				}
			}
			if len(waiters) == 0 {
				delete(hashWaiters, h)
			}
		}

		// wait for arrival or a new waiter
		select {
		case <-ctx.Done():
			return
		case w := <-mp.waiting:
			txs, ok := mp.LookupBatch(ctx, w.hash)
			if ok {
				w.reply <- txs
			} else {
				waiters, ok := hashWaiters[w.hash]
				if !ok {
					waiters = map[*waitingBatch]bool{}
					hashWaiters[w.hash] = waiters
				}
				waiters[w] = true
			}
		case arr := <-mp.arriving:
			for waiter := range hashWaiters[arr.hash] {
				waiter.reply <- arr.ptxs
			}
			delete(hashWaiters, arr.hash)
		}
	}
}

func (mp *BatchMempool) WaitBatch(ctx context.Context, hash proto.BatchHash) ([]*proto.ProposedTransaction, bool) {
	reply := make(chan []*proto.ProposedTransaction, 1)
	mp.waiting <- &waitingBatch{ctx: ctx, hash: hash, reply: reply}
	select {
	case <-ctx.Done():
		return nil, false
	case txs, ok := <-reply:
		return txs, ok
	}
}

func (mp *BatchMempool) supervise(_ context.Context) {
	for {
		mp.time.Sleep(mp.batchTTL)
		now := mp.time.Now()
		mp.lk.Lock()
		for k, b := range mp.batch {
			if now.Sub(b.created) > mp.batchTTL {
				delete(mp.batch, k)
			}
		}
		mp.lk.Unlock()
	}
}

type batch struct {
	created time.Time
	txs     []*proto.ProposedTransaction
}

func newBatch(created time.Time, txs []proto.ProposedTransaction) *batch {
	return &batch{
		created: created,
		txs:     protoutil.UnpackProposedTransactions(txs),
	}
}
