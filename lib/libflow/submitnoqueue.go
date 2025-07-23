package libflow

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/panjf2000/ants/v2"
)

type SubmitWorkerPoolNoQueue struct {
	ctx    context.Context
	cancel context.CancelFunc
	//
	pool *ants.Pool
	//
	utilization libstat.Counter
	submitLat   *libstat.LatencyMeter // time to submit
	waitLat     *libstat.LatencyMeter // time in queue
	workLat     *libstat.LatencyMeter // time working
}

func NewSubmitWorkerPoolNoQueue(
	ctx context.Context,
	numWorkers int,
) SubmitWorkerPool {

	ctx, cancel := context.WithCancel(ctx)

	p, err := ants.NewPool(
		numWorkers,
		ants.WithPanicHandler(func(p any) { panic(p) }),
	)
	must.NoError(ctx, err)

	return &SubmitWorkerPoolNoQueue{
		ctx:       ctx,
		cancel:    cancel,
		pool:      p,
		submitLat: libstat.NewLatency(libtime.RealTime{}),
		waitLat:   libstat.NewLatency(libtime.RealTime{}),
		workLat:   libstat.NewLatency(libtime.RealTime{}),
	}
}

func (x *SubmitWorkerPoolNoQueue) Stop() {
	x.cancel()
}

func (x *SubmitWorkerPoolNoQueue) Submit(task func()) {

	x.utilization.Add(float64(x.pool.Running()) / float64(x.pool.Cap()))

	submitSpan := x.submitLat.Start()
	waitSpan := x.waitLat.Start()

	err := x.pool.Submit(
		func() {
			waitSpan.Stop()
			workSpan := x.workLat.Start()
			task()
			workSpan.Stop()
		},
	)
	must.NoError(x.ctx, err)

	submitSpan.Stop()
}

func (x *SubmitWorkerPoolNoQueue) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		Utilization: x.utilization.Stats(),
		SubmitLat:   x.submitLat.Stats(),
		WaitLat:     x.waitLat.Stats(),
		WorkLat:     x.workLat.Stats(),
	}
}
