package libflow

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
)

type Flusher[T any] func(context.Context, []T)

type Buffer[T any] struct {
	time    libtime.Time
	flush   Flusher[T]
	maxSize int
	maxWait time.Duration
	ch      chan T
}

func NewBuffer[T any](
	ctx context.Context,
	time libtime.Time,
	flush Flusher[T],
	maxSize int,
	maxWait time.Duration,
) *Buffer[T] {
	must.Assert(ctx, maxSize > 0)
	must.Assert(ctx, maxWait > 0)
	buf := &Buffer[T]{
		time:    time,
		flush:   flush,
		maxSize: maxSize,
		maxWait: maxWait,
		ch:      make(chan T, maxSize),
	}
	go buf.loop(ctx)
	return buf
}

func (buf *Buffer[T]) Submit(ctx context.Context, xs ...T) {
	for _, x := range xs {
		buf.ch <- x
	}
}

func (buf *Buffer[T]) loop(ctx context.Context) {
	for {
		var pending []T
		wake := buf.time.After(buf.maxWait)
	fill:
		for len(pending) < buf.maxSize {
			select {
			case x := <-buf.ch:
				pending = append(pending, x)
			case <-ctx.Done():
				return
			case <-wake:
				break fill
			}
		}

		if len(pending) > 0 {
			buf.flush(ctx, pending)
			pending = pending[:0]
		}
	}
}
