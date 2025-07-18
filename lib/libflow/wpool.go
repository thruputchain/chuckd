package libflow

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/panjf2000/ants/v2"
)

type WorkerPool struct {
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
) *WorkerPool {

	ctx, cancel := context.WithCancel(ctx)

	p, err := ants.NewPool(numWorkers)
	must.NoError(ctx, err)

	x := &WorkerPool{
		ctx:       ctx,
		cancel:    cancel,
		queue:     make(chan submission, numWorkers),
		pool:      p,
		submitLat: libstat.NewLatencyWithReport(libtime.RealTime{}, label+":submit", time.Second*3),
		waitLat:   libstat.NewLatencyWithReport(libtime.RealTime{}, label+":wait", time.Second*3),
		workLat:   libstat.NewLatencyWithReport(libtime.RealTime{}, label+":work", time.Second*3),
	}

	go x.do()

	return x
}

func (x *WorkerPool) Stop() {
	x.cancel()
}

func (x *WorkerPool) Submit(task func()) {

	submitSpan := x.submitLat.Start()

	select {
	case <-x.ctx.Done():
	case x.queue <- submission{task: task, waitSpan: x.waitLat.Start()}:
	}

	submitSpan.Stop()
}

func (x *WorkerPool) do() {

	defer func() {
		for range x.queue {
		}
	}()

	for {
		select {
		case <-x.ctx.Done():
		case s := <-x.queue:
			s.waitSpan.Stop()

			err := x.pool.Submit(
				func() {
					workSpan := x.workLat.Start()
					s.task()
					workSpan.Stop()
				},
			)

			must.NoError(x.ctx, err)
		}
	}
}
