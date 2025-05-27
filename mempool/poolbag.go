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

type BagMempool struct {
	time     libtime.Time
	bagTTL   time.Duration
	arriving chan arrivingBag
	waiting  chan *waitingBag
	lk       sync.Mutex
	bag      map[proto.BagHash]*bag
}

type arrivingBag struct {
	hash proto.BagHash
	txs  []*proto.SignedTransaction
}

type waitingBag struct {
	ctx   context.Context
	hash  proto.BagHash
	reply chan<- []*proto.SignedTransaction
}

func NewBagMempoolReal(ctx context.Context, bagTTL time.Duration) *BagMempool {
	return NewBagMempool(ctx, libtime.RealTime{}, bagTTL)
}

func NewBagMempool(ctx context.Context, time libtime.Time, bagTTL time.Duration) *BagMempool {
	mp := &BagMempool{
		time:     time,
		bagTTL:   bagTTL,
		arriving: make(chan arrivingBag),
		waiting:  make(chan *waitingBag),
		bag:      map[proto.BagHash]*bag{},
	}
	go mp.supervise(ctx)
	go mp.wait(ctx)
	return mp
}

func (mp *BagMempool) BagCount() int {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	return len(mp.bag)
}

func (mp *BagMempool) AddBag(ctx context.Context, txs proto.BagTransactions) {
	hash := sha256.Sum256(xdrutil.MarshalXdr(&txs))
	mp.lk.Lock()
	b := newBag(mp.time.Now(), txs.Transactions)
	mp.bag[hash] = b
	mp.lk.Unlock()
	mp.arriving <- arrivingBag{hash: hash, txs: b.txs}
}

func (mp *BagMempool) LookupBag(ctx context.Context, bagHash proto.BagHash) ([]*proto.SignedTransaction, bool) {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	b, ok := mp.bag[bagHash]
	if ok {
		return b.txs, true
	} else {
		return nil, false
	}
}

func (mp *BagMempool) wait(ctx context.Context) {

	hashWaiters := map[proto.BagHash]map[*waitingBag]bool{}
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
			txs, ok := mp.LookupBag(ctx, w.hash)
			if ok {
				w.reply <- txs
			} else {
				waiters, ok := hashWaiters[w.hash]
				if !ok {
					waiters = map[*waitingBag]bool{}
					hashWaiters[w.hash] = waiters
				}
				waiters[w] = true
			}
		case arr := <-mp.arriving:
			for waiter := range hashWaiters[arr.hash] {
				waiter.reply <- arr.txs
			}
			delete(hashWaiters, arr.hash)
		}
	}
}

func (mp *BagMempool) WaitBag(ctx context.Context, hash proto.BagHash) ([]*proto.SignedTransaction, bool) {
	reply := make(chan []*proto.SignedTransaction, 1)
	mp.waiting <- &waitingBag{ctx: ctx, hash: hash, reply: reply}
	select {
	case <-ctx.Done():
		return nil, false
	case txs, ok := <-reply:
		return txs, ok
	}
}

func (mp *BagMempool) supervise(_ context.Context) {
	for {
		mp.time.Sleep(mp.bagTTL)
		now := mp.time.Now()
		mp.lk.Lock()
		for k, b := range mp.bag {
			if now.Sub(b.created) > mp.bagTTL {
				delete(mp.bag, k)
			}
		}
		mp.lk.Unlock()
	}
}

type bag struct {
	created time.Time
	txs     []*proto.SignedTransaction
}

func newBag(created time.Time, txs []proto.SignedTransaction) *bag {
	return &bag{
		created: created,
		txs:     protoutil.Unpack(txs),
	}
}
