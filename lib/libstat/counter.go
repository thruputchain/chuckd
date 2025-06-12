package libstat

import (
	"sync"
	"time"
)

type Counter struct {
	lk    sync.Mutex
	start time.Time
	sum   float64
	count int
}

func (x *Counter) Add(v float64) {
	x.lk.Lock()
	defer x.lk.Unlock()
	if x.start.IsZero() {
		x.start = time.Now()
		// discard v
	} else {
		x.sum += v
		x.count += 1
	}
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

func (x *Counter) SpeedPerSec() float64 {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.sum / time.Now().Sub(x.start).Seconds()
}
