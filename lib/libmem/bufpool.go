package libmem

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
)

type BufferPool struct {
	pool    sync.Pool
	lk      sync.Mutex
	numGet  int
	numMiss int
}

func NewBufferPool() *BufferPool {
	x := &BufferPool{}
	x.pool.New = func() any {
		x.lk.Lock()
		x.numMiss += 1
		x.lk.Unlock()
		return &borrowedBuffer{pool: x}
	}
	return x
}

func (x *BufferPool) Get() BorrowedBuffer {
	bb := x.pool.Get().(*borrowedBuffer)
	if !bb.borrowed.CompareAndSwap(false, true) {
		panic("borrowed buffer inside pool")
	}
	bb.Reset()

	x.lk.Lock()
	x.numGet += 1
	x.lk.Unlock()
	return bb
}

func (x *BufferPool) recycle(bb *borrowedBuffer) {
	x.pool.Put(bb)
}

func (x *BufferPool) StatString() string {
	x.lk.Lock()
	defer x.lk.Unlock()
	return fmt.Sprintf("num_get=%d num_miss=%d miss_ratio=%.3f%%",
		x.numGet, x.numMiss, 100*float64(x.numMiss)/float64(x.numGet))
}

//

type borrowedBuffer struct {
	borrowed atomic.Bool
	bytes.Buffer
	pool *BufferPool
}

func (x *borrowedBuffer) Recycle() {
	if !x.borrowed.CompareAndSwap(true, false) {
		panic("buffer not borrowed")
	}
	x.pool.recycle(x)
}
