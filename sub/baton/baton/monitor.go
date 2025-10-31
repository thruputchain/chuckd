package baton

import (
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
)

type Monitor struct {
	time           libtime.Time
	lk             sync.Mutex
	timeToCommit   *libstat.RingBuffer[time.Duration]
	timeToEnvelope *libstat.RingBuffer[time.Duration]
}

func NewMonitor(time libtime.Time) *Monitor {
	return &Monitor{time: time}
}

func (x *Monitor) Committed() {
	// XXX
}
