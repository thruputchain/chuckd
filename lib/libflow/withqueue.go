package libflow

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/panjf2000/ants/v2"
)

type WorkerPoolWithQueue struct {
	label  string
	ctx    context.Context
	cancel context.CancelFunc
	//
	queue chan submission
	pool  *ants.Pool
	//
	submitLat *libstat.LatencyMeter // time to submit
	waitLat   *libstat.LatencyMeter // time in queue
	workLat   *libstat.LatencyMeter // time working
}

type submission struct {
	task     func()
	waitSpan libstat.LatencyMeterSpan
}

func NewWorkerPool(
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

	x := &WorkerPoolWithQueue{
		label:     label,
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

func (x *WorkerPoolWithQueue) Stop() {
	x.cancel()
}

func (x *WorkerPoolWithQueue) Submit(task func()) {

	submitSpan := x.submitLat.Start()

	select {
	case <-x.ctx.Done():
	case x.queue <- submission{task: task, waitSpan: x.waitLat.Start()}:
	}

	submitSpan.Stop()
}

func (x *WorkerPoolWithQueue) do() {

	defer func() {
		for range x.queue {
		}
	}()

	i := 0
	for {
		if i%100 == 0 {
			liblog.Infof("%s total: %v, submit: %v, wait: %v, work: %v",
				x.label,
				libtime.FmtDur(x.submitLat.Average()+x.waitLat.Average()+x.workLat.Average()),
				x.submitLat, x.waitLat, x.workLat)
		}

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
		i += 1
	}
}
