package mempool

import (
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type batch struct {
	created time.Time
	txs     []*proto.ProposedTransaction
	//
	lk sync.Mutex
	// a batch is unpinned when the local validator will no longer need it for proposal or validation.
	pinned bool
}

func newBatch(txs []proto.ProposedTransaction) *batch {
	return &batch{
		created: time.Now(),
		txs:     protoutil.RepackageProposedTransactions(txs),
		pinned:  true,
	}
}

func (b *batch) Unpin() {
	b.lk.Lock()
	defer b.lk.Unlock()
	b.pinned = false
}

func (b *batch) Pinned() bool {
	b.lk.Lock()
	defer b.lk.Unlock()
	return b.pinned
}
