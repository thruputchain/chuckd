package libmem

import (
	"bytes"
	"sync/atomic"
)

type BorrowedBuffer struct {
	borrowed atomic.Bool
	bytes.Buffer
	pool *BufferPool
}

func (x *BorrowedBuffer) Recycle() {
	if !x.borrowed.CompareAndSwap(true, false) {
		panic("buffer not borrowed")
	}
	x.pool.recycle(x)
}
