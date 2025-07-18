package libstat

import (
	"math"
	"sync"
	"time"
)

type Counter struct {
	lk    sync.Mutex
	start time.Time
	sum   float64
	sum2  float64
	min   float64
	max   float64
	count int
}

func (x *Counter) Add(v float64) {
	x.lk.Lock()
	defer x.lk.Unlock()
	if x.start.IsZero() {
		x.start = time.Now()
		x.min = v
		x.max = v
		// discard first v for purposes of sum, average, stddev
	} else {
		x.min = min(x.min, v)
		x.max = max(x.max, v)
		x.sum += v
		x.sum2 += v * v
		x.count += 1
	}
}

func (x *Counter) Min() float64 {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.min
}

func (x *Counter) Max() float64 {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.max
}

func (x *Counter) Sum() float64 {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.sum
}

func (x *Counter) Average() float64 {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.sum / float64(x.count)
}

func (x *Counter) Stddev() float64 {
	x.lk.Lock()
	defer x.lk.Unlock()

	avg := x.sum / float64(x.count)
	avg2 := x.sum2 / float64(x.count)

	return math.Sqrt(avg2 - avg*avg)
}

func (x *Counter) SpeedPerSec() float64 {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.sum / time.Since(x.start).Seconds()
}
