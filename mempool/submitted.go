package mempool

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type SubmittedMempool struct {
	time     libtime.Time
	txTTL    time.Duration
	arriving chan *arrivingTxs
	waiting  chan *waitingTxs
	m        xdrutil.XdrMarshaler
	lk       sync.Mutex
	tx       map[proto.SignedTransactionHash]*transaction
}

type transaction struct {
	created time.Time
	proto   *proto.SignedTransaction
}

type arrivingTxs struct {
	txs []*proto.SignedTransaction
}

type waitingTxs struct {
	ctx    context.Context
	hashes []proto.SignedTransactionHash
	reply  chan<- []*proto.SignedTransaction
}

func NewTransactionMempoolReal(ctx context.Context, txTTL time.Duration) *SubmittedMempool {
	return NewTransactionMempool(ctx, libtime.RealTime{}, txTTL)
}

func NewTransactionMempool(ctx context.Context, time libtime.Time, txTTL time.Duration) *SubmittedMempool {
	mp := &SubmittedMempool{
		time:     time,
		txTTL:    txTTL,
		arriving: make(chan *arrivingTxs),
		waiting:  make(chan *waitingTxs),
		m:        xdrutil.NewXdrMarshaler(),
		tx:       map[proto.SignedTransactionHash]*transaction{},
	}
	go mp.supervise(ctx)
	go mp.wait(ctx)
	return mp
}

func (mp *SubmittedMempool) AddTxs(ctx context.Context, txs ...*proto.SignedTransaction) {
	mp.lk.Lock()
	for _, tx := range txs {
		mp.addTx(ctx, tx)
	}
	mp.lk.Unlock()
	mp.arriving <- &arrivingTxs{txs: txs}
}

func (mp *SubmittedMempool) addTx(_ context.Context, tx *proto.SignedTransaction) {
	mp.tx[protoutil.HashSignedTx(mp.m, tx)] = &transaction{created: mp.time.Now(), proto: tx}
}

func (mp *SubmittedMempool) TxCount() int {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	return len(mp.tx)
}

func (mp *SubmittedMempool) ListTxs(ctx context.Context) []*proto.SignedTransaction {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	txs := []*proto.SignedTransaction{}
	for _, tx := range mp.tx {
		txs = append(txs, tx.proto)
	}
	return txs
}

func (mp *SubmittedMempool) RemoveTx(ctx context.Context, k proto.SignedTransactionHash) {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	delete(mp.tx, k)
}

func (mp *SubmittedMempool) LookupTxs(ctx context.Context, hashes []proto.SignedTransactionHash) ([]*proto.SignedTransaction, bool) {
	txs := make([]*proto.SignedTransaction, len(hashes))
	mp.lk.Lock()
	defer mp.lk.Unlock()
	for i, h := range hashes {
		tx, ok := mp.tx[h]
		if !ok {
			return nil, false
		}
		txs[i] = tx.proto
	}
	return txs, true
}

func (mp *SubmittedMempool) WaitTxs(ctx context.Context, hashes []proto.SignedTransactionHash) ([]*proto.SignedTransaction, bool) {
	reply := make(chan []*proto.SignedTransaction, 1)
	mp.waiting <- &waitingTxs{ctx: ctx, hashes: hashes, reply: reply}
	select {
	case <-ctx.Done():
		return nil, false
	case txs, ok := <-reply:
		return txs, ok
	}
}

func (mp *SubmittedMempool) wait(ctx context.Context) {

	waiters := map[*waitingTxs]bool{}
	defer func() {
		for waiter := range waiters {
			close(waiter.reply)
		}
	}()

	for {
		// cleanup cancelled waiters
		for waiter := range waiters {
			select {
			case <-waiter.ctx.Done():
				delete(waiters, waiter)
			default:
			}
		}

		// wait for arrival or a new waiter
		select {
		case <-ctx.Done():
			return
		case w := <-mp.waiting:
			txs, ok := mp.LookupTxs(ctx, w.hashes)
			if ok {
				w.reply <- txs
			} else {
				liblog.Infof("waiting for txs: %v", form.Display(w.hashes))
				waiters[w] = true
			}
		case <-mp.arriving:
			for w := range waiters {
				txs, ok := mp.LookupTxs(ctx, w.hashes)
				if ok {
					w.reply <- txs
					delete(waiters, w)
				}
			}
		}
	}
}

func (mp *SubmittedMempool) supervise(_ context.Context) {
	for {
		mp.time.Sleep(mp.txTTL)
		now := mp.time.Now()
		mp.lk.Lock()
		for k, tx := range mp.tx {
			if now.Sub(tx.created) > mp.txTTL {
				delete(mp.tx, k)
			}
		}
		mp.lk.Unlock()
	}
}
