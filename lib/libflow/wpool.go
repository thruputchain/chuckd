package libflow

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/libstat"
)

type SubmitWorkerPool interface {
	Submit(func())
	Stop()
	Stats() WorkerPoolStats
}

func NewSubmitWorkerPool(ctx context.Context, numWorkers int) SubmitWorkerPool {
	return NewSubmitWorkerPoolWithQueue(ctx, numWorkers)
}

type InvokeWorkerPool[T any] interface {
	Invoke(T)
	Stop()
	Stats() WorkerPoolStats
}

func NewInvokeWorkerPool[T any](ctx context.Context, f func(T), numWorkers int) InvokeWorkerPool[T] {
	return NewInvokeWorkerPoolWithQueue[T](ctx, f, numWorkers)
}

// stats

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
