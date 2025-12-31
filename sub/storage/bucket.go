package storage

import (
	"context"
	"slices"
	"sync"
)

type Bucket[V any, Q any] struct {
	pause  sync.Mutex //XXX: pause flushing, called from app
	cond   *sync.Cond
	lk     sync.Mutex
	things []V
}

func NewBucket[V any, Q any]() *Bucket[V, Q] {
	x := &Bucket[V, Q]{}
	x.cond = sync.NewCond(&x.lk)
	return x
}

func (x *Bucket[V, Q]) Start(
	ctx context.Context,
	flush func(context.Context, []V),
	same func(V, V) bool,
) {
	go func() {
		x.lk.Lock()
		defer x.lk.Unlock()
		for {
			if len(x.things) > 0 {
				snapshot := slices.Clone(x.things)
				x.lk.Unlock()

				flush(ctx, snapshot)

				x.lk.Lock()
				x.scrub(snapshot, same)
			}
			x.cond.Wait()
		}
	}()
}

func (x *Bucket[V, Q]) scrub(v []V, same func(V, V) bool) {
	XXX
}

func (x *Bucket[V, Q]) Len() int {
	x.lk.Lock()
	defer x.lk.Unlock()

	return len(x.things)
}

func (x *Bucket[V, Q]) Add(v V) {
	x.lk.Lock()
	defer x.lk.Unlock()

	defer x.cond.Signal()

	x.things = append(x.things, v)
}

func (x *Bucket[V, Q]) FindAll(query Q, match func(Q, V) bool) []V {
	x.lk.Lock()
	defer x.lk.Unlock()

	found := []V{}
	for _, v := range x.things {
		if match(query, v) {
			found = append(found, v)
		}
	}
	return found
}

func (x *Bucket[V, Q]) FindOne(query Q, match func(Q, V) bool) *V {
	x.lk.Lock()
	defer x.lk.Unlock()

	for _, v := range x.things {
		if match(query, v) {
			return &v
		}
	}
	return nil
}

func (x *Bucket[V, Q]) Reduce(reduce func(prev *V, next V) *V) *V {
	x.lk.Lock()
	defer x.lk.Unlock()

	var reduced *V
	for _, v := range x.things {
		reduced = reduce(reduced, v)
	}
	return reduced
}
