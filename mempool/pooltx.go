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

type TransactionMempool struct {
	time  libtime.Time
	txTTL time.Duration
	lk    sync.Mutex
	tx    map[proto.SHA256Hash]*transaction
}

type transaction struct {
	created time.Time
	proto   *proto.SignedTransaction
}

func NewTransactionMempoolReal(ctx context.Context, txTTL time.Duration) *TransactionMempool {
	return NewTransactionMempool(ctx, libtime.RealTime{}, txTTL)
}

func NewTransactionMempool(ctx context.Context, time libtime.Time, txTTL time.Duration) *TransactionMempool {
	mp := &TransactionMempool{
		time:  time,
		txTTL: txTTL,
		tx:    map[proto.SHA256Hash]*transaction{},
	}
	go mp.supervise(ctx)
	return mp
}

func (mp *TransactionMempool) AddTxs(ctx context.Context, tx []proto.SignedTransaction) {
	for _, tx := range tx {
		mp.AddTx(ctx, &tx)
	}
}

func (mp *TransactionMempool) AddTx(ctx context.Context, tx *proto.SignedTransaction) {
	k := sha256.Sum256(xdrutil.MarshalXdr(tx))
	mp.lk.Lock()
	defer mp.lk.Unlock()
	mp.tx[k] = &transaction{created: mp.time.Now(), proto: tx}
}

func (mp *TransactionMempool) ListTxs(ctx context.Context) []*proto.SignedTransaction {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	txs := []*proto.SignedTransaction{}
	for _, tx := range mp.tx {
		txs = append(txs, tx.proto)
	}
	return txs
}

func (mp *TransactionMempool) RemoveTx(ctx context.Context, k proto.SignedTransactionHash) {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	delete(mp.tx, k)
}

func (mp *TransactionMempool) supervise(_ context.Context) {
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
