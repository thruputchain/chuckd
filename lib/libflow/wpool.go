package libflow

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/panjf2000/ants/v2"
)

type WorkerXXX interface {
	Submit(func())
	Stop()
}

type WorkerPool struct {
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
) *WorkerPool {

	ctx, cancel := context.WithCancel(ctx)

	p, err := ants.NewPool(numWorkers)
	must.NoError(ctx, err)

	x := &WorkerPool{
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
