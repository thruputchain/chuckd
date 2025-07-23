package libflow

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/panjf2000/ants/v2"
)

type InvokeWorkerPoolWithQueue[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	//
	queue chan invocation[T]
	pool  *ants.PoolWithFuncGeneric[argWithSpan[T]]
	//
	utilization libstat.Counter
	submitLat   *libstat.LatencyMeter // time to submit
	waitLat     *libstat.LatencyMeter // time in queue
	workLat     *libstat.LatencyMeter // time working
}

type invocation[T any] struct {
	arg      T
	waitSpan libstat.LatencyMeterSpan
}

type argWithSpan[T any] struct {
	arg      T
	waitSpan libstat.LatencyMeterSpan
}

func NewInvokeWorkerPoolWithQueue[T any](
	ctx context.Context,
	f func(T),
	numWorkers int,
) InvokeWorkerPool[T] {

	ctx, cancel := context.WithCancel(ctx)

	x := &InvokeWorkerPoolWithQueue[T]{
		ctx:       ctx,
		cancel:    cancel,
		queue:     make(chan invocation[T], numWorkers),
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

	go x.do()

	return x
}

func (x *InvokeWorkerPoolWithQueue[T]) Stop() {
	x.cancel()
}

func (x *InvokeWorkerPoolWithQueue[T]) Invoke(arg T) {

	x.utilization.Add(float64(x.pool.Running()) / float64(x.pool.Cap()))

	submitSpan := x.submitLat.Start()

	select {
	case <-x.ctx.Done():
	case x.queue <- invocation[T]{arg: arg, waitSpan: x.waitLat.Start()}:
	}

	submitSpan.Stop()
}

func (x *InvokeWorkerPoolWithQueue[T]) do() {

	defer func() {
		for range x.queue {
		}
	}()

	for {
		select {
		case <-x.ctx.Done():
			return
		case s := <-x.queue:

			err := x.pool.Invoke(argWithSpan[T]{arg: s.arg, waitSpan: s.waitSpan})
			must.NoError(x.ctx, err)

		}
	}
}

func (x *InvokeWorkerPoolWithQueue[T]) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		Utilization: x.utilization.Stats(),
		SubmitLat:   x.submitLat.Stats(),
		WaitLat:     x.waitLat.Stats(),
		WorkLat:     x.workLat.Stats(),
	}
}
