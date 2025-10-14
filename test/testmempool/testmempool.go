package testmempool

import (
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestMempool struct {
	m    xdrutil.XdrMarshaler
	lk   sync.Mutex
	pool map[proto.SignedTransactionHash]*proto.SignedTransaction
}

func NewTestMempool() *TestMempool {
	return &TestMempool{
		m:    xdrutil.NewXdrMarshaler(),
		pool: map[proto.SignedTransactionHash]*proto.SignedTransaction{},
	}
}

func (x *TestMempool) AddTxs(txs []proto.SignedTransaction) {
	x.lk.Lock()
	defer x.lk.Unlock()
	for _, tx := range txs {
		x.pool[protoutil.HashSignedTx(x.m, &tx)] = &tx
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
	hashes []proto.SignedTransactionHash,
) (found []*proto.SignedTransaction, missing []proto.SignedTransactionHash) {
	x.lk.Lock()
	defer x.lk.Unlock()
	found = make([]*proto.SignedTransaction, len(hashes))
	missing = []proto.SignedTransactionHash{}
	for i, h := range hashes {
		var ok bool
		found[i], ok = x.pool[h]
		if !ok {
			missing = append(missing, h)
		}
	}
	if len(missing) > 0 {
		return nil, missing
	}
	return found, nil
}

func (x *TestMempool) RemoveTxs(hashes []proto.SignedTransactionHash) {
	x.lk.Lock()
	defer x.lk.Unlock()
	for _, h := range hashes {
		delete(x.pool, h)
	}
}
