package libflow

import (
	"fmt"

	"github.com/groundhogchain/chuckd/lib/libstat"
)

type WorkerPool interface {
	Submit(func())
	Stop()
	Stats() WorkerPoolStats
}

type WorkerPoolStats struct {
	Utilization libstat.CounterStats
	SubmitLat   libstat.LatencyStats
	WaitLat     libstat.LatencyStats
	WorkLat     libstat.LatencyStats
}

func (x WorkerPoolStats) Summarize() string {
	return fmt.Sprintf(
		"utilization: %.2f%%, submit: %s, wait: %s, work: %s",
		x.Utilization.Average*100,
		x.SubmitLat.Summarize(),
		x.WaitLat.Summarize(),
		x.WorkLat.Summarize(),
	)
}
