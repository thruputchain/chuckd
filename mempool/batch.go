package mempool

import (
	"sync"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

type batch struct {
	header  proto.BatchHeader
	created time.Time
	txs     []proto.ProposedTransaction
	//
	lk sync.Mutex
	// a batch is unpinned when the local validator will no longer need it for proposal or validation.
	pinned bool
}

func newBatch(header proto.BatchHeader, txs []proto.ProposedTransaction) *batch {
	return &batch{
		header:  header,
		created: time.Now(),
		txs:     txs,
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
