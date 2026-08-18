package testmempool

import (
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestMempool struct {
	lk   sync.Mutex
	pool map[proto.TransactionHash]*proto.SignedTransaction
}

func NewTestMempool() *TestMempool {
	return &TestMempool{
		pool: map[proto.TransactionHash]*proto.SignedTransaction{},
	}
}

func (x *TestMempool) AddTxs(txs []proto.SignedTransaction) {
	x.lk.Lock()
	defer x.lk.Unlock()
	for _, tx := range txs {
		x.pool[protoutil.HashSignedTx(&tx)] = &tx
	}
}

func (x *TestMempool) DrainTxs() []*proto.SignedTransaction {
	x.lk.Lock()
	defer x.lk.Unlock()
	r := make([]*proto.SignedTransaction, 0, len(x.pool))
	for _, tx := range x.pool {
		r = append(r, tx)
	}
	return r
}

func (x *TestMempool) LookupTxs(
	hashes []proto.TransactionHash,
) (
	found []proto.SignedTransaction,
	missing []proto.TransactionHash,
	mixed proto.OptSignedTransactionBatch,
) {
	x.lk.Lock()
	defer x.lk.Unlock()
	found = make([]proto.SignedTransaction, len(hashes))
	missing = []proto.TransactionHash{}
	mixed = make(proto.OptSignedTransactionBatch, len(hashes))
	for i, h := range hashes {
		got, ok := x.pool[h]
		if ok {
			found[i] = *got
			mixed[i] = got
		} else {
			missing = append(missing, h)
		}
	}
	if len(missing) > 0 {
		return nil, missing, mixed
	}
	return found, nil, mixed
}

func (x *TestMempool) RemoveTxs(hashes []proto.TransactionHash) {
	x.lk.Lock()
	defer x.lk.Unlock()
	for _, h := range hashes {
		delete(x.pool, h)
	}
}

func (x *TestMempool) Len() int {
	x.lk.Lock()
	defer x.lk.Unlock()
	return len(x.pool)
}
