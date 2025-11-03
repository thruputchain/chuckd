package libstat

import (
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libtime"
)

type RingBuffer[T any] struct {
	k    int64
	data []T
}

func NewRingBuffer[T any](n int) *RingBuffer[T] {
	return &RingBuffer[T]{data: make([]T, n)}
}

func (x *RingBuffer[T]) Add(t T) {
	x.data[x.k] = t
	x.k += 1
}

func (x *RingBuffer[T]) Count() int {
	return int(x.k)
}

func (x *RingBuffer[T]) Expose() []T {
	if x.k > int64(len(x.data)) {
		return x.data
	}
	return x.data[:x.k]
}

type MovingWindowDurationCounter struct {
	lk      sync.Mutex
	time    libtime.Time
	last    time.Time
	samples *RingBuffer[time.Duration]
}

func NewMovingWindowDurationCounter(t libtime.Time, n int) *MovingWindowDurationCounter {
	return &MovingWindowDurationCounter{
		time:    t,
		samples: NewRingBuffer[time.Duration](n),
	}
}

func (x *MovingWindowDurationCounter) SinceLast() *time.Duration {
	if x.last.IsZero() {
		return nil
	}
	elapsed := x.time.Now().Sub(x.last)
	return &elapsed
}

func (x *MovingWindowDurationCounter) Tick() {
	x.lk.Lock()
	defer x.lk.Unlock()
	if x.last.IsZero() {
		x.last = x.time.Now()
	} else {
		x.samples.Add(x.time.Now().Sub(x.last))
	}
}

func (x *MovingWindowDurationCounter) Count() int {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.samples.Count()
}

func (x *MovingWindowDurationCounter) Counter() *Counter {
	x.lk.Lock()
	defer x.lk.Unlock()
	exp := x.samples.Expose()
	samples := make([]float64, len(exp))
	for i, exp := range exp {
		samples[i] = float64(exp.Nanoseconds())
	}
	return NewCounterFromSamples(samples)
}

func (x *MovingWindowDurationCounter) MeanStddev() (time.Duration, time.Duration) {
	c := x.Counter()
	return time.Nanosecond * time.Duration(c.Average()),
		time.Nanosecond * time.Duration(c.Stddev())
}
