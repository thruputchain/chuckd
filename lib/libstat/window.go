package libstat

import (
	"sync"
	"time"
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
	last    time.Time
	samples *RingBuffer[time.Duration]
}

func NewMovingWindowDurationCounter(n int) *MovingWindowDurationCounter {
	return &MovingWindowDurationCounter{samples: NewRingBuffer[time.Duration](n)}
}

var zeroTime time.Time

func (x *MovingWindowDurationCounter) Tick() {
	x.lk.Lock()
	defer x.lk.Unlock()
	if x.last.Equal(zeroTime) {
		x.last = time.Now()
	} else {
		x.samples.Add(time.Since(x.last))
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
