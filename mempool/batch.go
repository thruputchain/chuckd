package mempool

import (
	"sync"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

type batch struct {
	header  proto.BatchHeader
	started time.Time
	ended   time.Time
	lk      sync.Mutex
	txs     []proto.Transaction
}

func newBatch(header proto.BatchHeader) *batch {
	return &batch{
		header:  header,
		started: time.Now(),
	}
}

// IsComplete returns true when the hash of the set of known transactions equals the batch ID.
func (b *batch) IsComplete() bool {
	panic("XXX")
}
