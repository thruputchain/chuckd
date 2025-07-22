package libflow

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/panjf2000/ants/v2"
)

type WorkerPoolNoQueue struct {
	label  string
	ctx    context.Context
	cancel context.CancelFunc
	//
	pool *ants.Pool
	//
	utilization libstat.Counter
	submitLat   *libstat.LatencyMeter // time to submit
	waitLat     *libstat.LatencyMeter // time in queue
	workLat     *libstat.LatencyMeter // time working
	completed   libstat.Counter
}

func NewWorkerPoolNoQueue(
	ctx context.Context,
	label string,
	numWorkers int,
) WorkerPool {

	ctx, cancel := context.WithCancel(ctx)

	p, err := ants.NewPool(
		numWorkers,
		ants.WithPanicHandler(func(p any) { panic(p) }),
	)
	must.NoError(ctx, err)

	return &WorkerPoolNoQueue{
		label:     label,
		ctx:       ctx,
		cancel:    cancel,
		pool:      p,
		submitLat: libstat.NewLatency(libtime.RealTime{}),
		waitLat:   libstat.NewLatency(libtime.RealTime{}),
		workLat:   libstat.NewLatency(libtime.RealTime{}),
	}
}

func (x *WorkerPoolNoQueue) Stop() {
	x.cancel()
}

func (x *WorkerPoolNoQueue) Submit(task func()) {

	x.utilization.Add(float64(x.pool.Running()) / float64(x.pool.Cap()))

	submitSpan := x.submitLat.Start()
	waitSpan := x.waitLat.Start()

	err := x.pool.Submit(
		func() {
			waitSpan.Stop()
			workSpan := x.workLat.Start()
			task()
			workSpan.Stop()

			if k := x.completed.Add(1); k%100 == 0 && k > 0 {
				liblog.Infof("%s total: %v, utilization: %.2f%%±%.2f%%, submit: %v, wait: %v, work: %v",
					x.label,
					libtime.FmtDur(x.submitLat.Average()+x.waitLat.Average()+x.workLat.Average()),
					x.utilization.Average()*100, x.utilization.Stddev()*100,
					x.submitLat, x.waitLat, x.workLat)
			}
		},
	)
	must.NoError(x.ctx, err)

	submitSpan.Stop()
}
