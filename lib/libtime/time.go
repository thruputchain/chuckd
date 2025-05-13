package libtime

import (
	"sync"
	"time"
)

type Time interface {
	Now() time.Time
	Sleep(time.Duration)
}

type RealTime struct{}

func (RealTime) Now() time.Time {
	return time.Now()
}

func (RealTime) Sleep(d time.Duration) {
	time.Sleep(d)
}

type TestTime struct {
	lk    sync.Mutex
	cond  *sync.Cond
	now   time.Time
	sleep chan time.Duration
}

func NewTestTime(now time.Time) *TestTime {
	tt := &TestTime{
		now:   now,
		sleep: make(chan time.Duration),
	}
	tt.cond = sync.NewCond(&tt.lk)
	return tt
}

func (tt *TestTime) Now() time.Time {
	tt.lk.Lock()
	defer tt.lk.Unlock()
	return tt.now
}

func (tt *TestTime) PassTime(d time.Duration) {
	tt.lk.Lock()
	defer tt.lk.Unlock()
	tt.now = tt.now.Add(d)
	tt.cond.Broadcast()
}

func (tt *TestTime) Sleep(d time.Duration) {
	tt.lk.Lock()
	defer tt.lk.Unlock()
	asleep := tt.now
	for tt.now.Sub(asleep) < d {
		tt.cond.Wait()
	}
}
