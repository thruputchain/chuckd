package libflow

import "github.com/groundhogchain/chuckd/lib/libstat"

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
