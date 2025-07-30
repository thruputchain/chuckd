package libmem

import (
	"io"
	"sync/atomic"
)

// value

type BorrowedValue[T any] interface {
	Value() T
	Recycle()
}

type borrowedValue[T any] struct {
	borrowed atomic.Bool
	value    T
	recycle  func()
}

func BorrowValue[T any](v T, recycle func()) BorrowedValue[T] {
	bv := &borrowedValue[T]{
		value:   v,
		recycle: recycle,
	}
	bv.borrowed.Store(true)
	return bv
}
func (bv *borrowedValue[T]) Value() T {
	return bv.value
}

func (bv *borrowedValue[T]) Recycle() {
	if !bv.borrowed.CompareAndSwap(true, false) {
		panic("value not borrowed")
	}
	bv.recycle()
}

// bytes

type BorrowedBytes interface {
	Bytes() []byte
	Len() int
	Recycle()
}

func BorrowBytes(b []byte) BorrowedBytes {
	return borrowedBytes(b)
}

type borrowedBytes []byte

func (bb borrowedBytes) Bytes() []byte {
	return bb
}

func (bb borrowedBytes) Len() int {
	return len(bb)
}

func (bb borrowedBytes) Recycle() {}

// buffer

type BorrowedBuffer interface {
	BorrowedBytes
	io.Reader
	io.Writer
	Reset()
}
