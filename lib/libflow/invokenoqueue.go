package libflow

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/panjf2000/ants/v2"
)

type InvokeWorkerPoolNoQueue[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	//
	pool *ants.PoolWithFuncGeneric[argWithSpan[T]]
	//
	utilization libstat.Counter
	submitLat   *libstat.LatencyMeter // time to submit
	waitLat     *libstat.LatencyMeter // time in queue
	workLat     *libstat.LatencyMeter // time working
}

func NewInvokeWorkerPoolNoQueue[T any](
	ctx context.Context,
	f func(T),
	numWorkers int,
) InvokeWorkerPool[T] {

	ctx, cancel := context.WithCancel(ctx)

	x := &InvokeWorkerPoolNoQueue[T]{
		ctx:       ctx,
		cancel:    cancel,
		submitLat: libstat.NewLatency(libtime.RealTime{}),
		waitLat:   libstat.NewLatency(libtime.RealTime{}),
		workLat:   libstat.NewLatency(libtime.RealTime{}),
	}

	p, err := ants.NewPoolWithFuncGeneric[argWithSpan[T]](
		numWorkers,
		func(arg argWithSpan[T]) {
			arg.waitSpan.Stop()
			workSpan := x.workLat.Start()
			f(arg.arg)
			workSpan.Stop()
		},
		ants.WithPanicHandler(func(p any) { panic(p) }),
	)
	must.NoError(ctx, err)

	x.pool = p

	return x
}

func (x *InvokeWorkerPoolNoQueue[T]) Stop() {
	x.cancel()
}

func (x *InvokeWorkerPoolNoQueue[T]) Invoke(arg T) {

	x.utilization.Add(float64(x.pool.Running()) / float64(x.pool.Cap()))

	submitSpan := x.submitLat.Start()
	waitSpan := x.waitLat.Start()

	err := x.pool.Invoke(argWithSpan[T]{arg: arg, waitSpan: waitSpan})
	must.NoError(x.ctx, err)

	submitSpan.Stop()
}

func (x *InvokeWorkerPoolNoQueue[T]) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		Utilization: x.utilization.Stats(),
		SubmitLat:   x.submitLat.Stats(),
		WaitLat:     x.waitLat.Stats(),
		WorkLat:     x.workLat.Stats(),
	}
}
