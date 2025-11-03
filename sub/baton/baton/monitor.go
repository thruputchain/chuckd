package baton

import (
	"sync"

	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
)

type Monitor struct {
	time              libtime.Time
	lk                sync.Mutex
	timeToCommit      *libstat.MovingWindowDurationCounter
	timeToExternalize *libstat.MovingWindowDurationCounter
}

const MonitorMemoryLen = 1000

func NewMonitor(time libtime.Time) *Monitor {
	return &Monitor{
		time:              time,
		timeToCommit:      libstat.NewMovingWindowDurationCounter(time, MonitorMemoryLen),
		timeToExternalize: libstat.NewMovingWindowDurationCounter(time, MonitorMemoryLen),
	}
}

func (x *Monitor) Externalized(slot int64) {
	// XXX
	x.timeToExternalize.Tick()
}

func (x *Monitor) Committed(slot int64) {
	// XXX
	x.timeToCommit.Tick()
}
