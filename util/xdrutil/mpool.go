package xdrutil

import (
	"bytes"
	"fmt"
	"sync"
)

type BufferPool struct {
	pool sync.Pool
	// stats
	lk      sync.Mutex
	numGet  int
	numMiss int
	cap     [256]int
	capPtr  int
}

func NewBufferPool() *BufferPool {
	x := &BufferPool{}
	x.pool.New = func() any {
		x.lk.Lock()
		x.numMiss += 1
		x.lk.Unlock()
		return &bytes.Buffer{}
	}
	return x
}

func (x *BufferPool) Get() *bytes.Buffer {
	buf := x.pool.Get().(*bytes.Buffer)
	buf.Reset()

	x.lk.Lock()
	x.numGet += 1
	if buf.Cap() > 0 {
		x.cap[x.capPtr%len(x.cap)] = buf.Cap()
		x.capPtr += 1
	}
	x.lk.Unlock()
	return buf
}

func (x *BufferPool) Recycle(b *bytes.Buffer) {
	x.pool.Put(b)
}

func (x *BufferPool) StatString() string {
	x.lk.Lock()
	defer x.lk.Unlock()
	return fmt.Sprintf("num_get=%d num_miss=%d miss_ratio=%.3f%% caps=%v",
		x.numGet, x.numMiss, 100*float64(x.numMiss)/float64(x.numGet), x.cap)
}
