package libflow

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/panjf2000/ants/v2"
)

type SubmitWorkerPoolWithQueue struct {
	ctx    context.Context
	cancel context.CancelFunc
	//
	queue chan submission
	pool  *ants.Pool
	//
	utilization libstat.Counter
	submitLat   *libstat.LatencyMeter // time to submit
	waitLat     *libstat.LatencyMeter // time in queue
	workLat     *libstat.LatencyMeter // time working
}

type submission struct {
	task     func()
	waitSpan libstat.LatencyMeterSpan
}

func NewSubmitWorkerPoolWithQueue(
	ctx context.Context,
	numWorkers int,
) SubmitWorkerPool {

	ctx, cancel := context.WithCancel(ctx)

	p, err := ants.NewPool(
		numWorkers,
		ants.WithPanicHandler(func(p any) { panic(p) }),
	)
	must.NoError(ctx, err)

	x := &SubmitWorkerPoolWithQueue{
		ctx:       ctx,
		cancel:    cancel,
		queue:     make(chan submission, numWorkers),
		pool:      p,
		submitLat: libstat.NewLatency(libtime.RealTime{}),
		waitLat:   libstat.NewLatency(libtime.RealTime{}),
		workLat:   libstat.NewLatency(libtime.RealTime{}),
	}

	go x.do()

	return x
}

func (x *SubmitWorkerPoolWithQueue) Stop() {
	x.cancel()
}

func (x *SubmitWorkerPoolWithQueue) Submit(task func()) {

	x.utilization.Add(float64(x.pool.Running()) / float64(x.pool.Cap()))

	submitSpan := x.submitLat.Start()

	select {
	case <-x.ctx.Done():
	case x.queue <- submission{task: task, waitSpan: x.waitLat.Start()}:
	}

	submitSpan.Stop()
}

func (x *SubmitWorkerPoolWithQueue) do() {

	defer func() {
		for range x.queue {
		}
	}()

	for {
		select {
		case <-x.ctx.Done():
			return
		case s := <-x.queue:

			err := x.pool.Submit(
				func() {
					s.waitSpan.Stop()
					workSpan := x.workLat.Start()
					s.task()
					workSpan.Stop()
				},
			)

			must.NoError(x.ctx, err)
		}
	}
}

func (x *SubmitWorkerPoolWithQueue) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		Utilization: x.utilization.Stats(),
		SubmitLat:   x.submitLat.Stats(),
		WaitLat:     x.waitLat.Stats(),
		WorkLat:     x.workLat.Stats(),
	}
}
