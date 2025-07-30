package libmem

import (
	"sync"
	"sync/atomic"
)

// The caller is responsible for adding 1 to the wait group before calling BorrowOnWaitGroup.
func BorrowOnWaitGroup[T any](wg *sync.WaitGroup, value T) BorrowedValue[T] {
	bv := &wgValue[T]{wg: wg, v: value}
	bv.borrowed.Store(true)
	return bv
}

type wgValue[T any] struct {
	borrowed atomic.Bool
	wg       *sync.WaitGroup
	v        T
}

func (x *wgValue[T]) Value() T {
	return x.v
}

func (x *wgValue[T]) Recycle() {
	if !x.borrowed.CompareAndSwap(true, false) {
		panic("bug")
	}
	x.wg.Done()
}
