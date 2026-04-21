package coordinator_bench

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (s *CoordinatorBenchSystem) Start(
	ctx context.Context,
	opcfg coordinator.OperationalConfig,
	numCommits int, // stop after this many commits
	ready chan<- struct{},
) {
	// record the beginning of the benchmark
	t0 := time.Now()

	// start scheduler rpc, needed by coordinator system
	s.CoordinatorBenchCluster.StartScheduler(ctx)

	// start coordinator
	ctl := s.System.Start(ctx, opcfg)
	ctl.SignalGenesisIsReady()

	// connect the test scheduler to the coordinator
	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, s.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	s.sched.SetCoordScheduler(coordSchedConn)

	s.CoordinatorBenchCluster.StartPump(ctx)

	// wait for numCommits and signal ready
	go func() {
		for i := 0; i < numCommits; i++ {
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			default:
			}
			if s.notify.ReceiveCommitted(ctx) == nil {
				return
			}
		}
		benchDuration := time.Since(t0)

		liblog.Infof(ctx, "coordinator benchmark completed in %s, %f ms/block",
			benchDuration, float64(benchDuration.Milliseconds())/float64(numCommits))
		close(ready)

		// keep running
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			default:
			}
			if s.notify.ReceiveCommitted(ctx) == nil {
				return
			}
		}
	}()
}

func (system *CoordinatorBenchSystem) startAndWaitHeight(
	ctx context.Context,
	opcfg coordinator.OperationalConfig,
	targetHeight proto.BlockHeight,
	ready chan<- struct{},
	onCommit func(*batonkit.CommittableBlock),
) {
	t0 := time.Now()

	system.CoordinatorBenchCluster.StartScheduler(ctx)

	ctl := system.System.Start(ctx, opcfg)
	ctl.SignalGenesisIsReady()

	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, system.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	system.sched.SetCoordScheduler(coordSchedConn)

	system.CoordinatorBenchCluster.StartPump(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-system.stop:
				return
			default:
			}

			cb := system.notify.ReceiveCommitted(ctx)
			if cb == nil {
				return
			}

			if onCommit != nil {
				onCommit(cb)
			}
			if cb.BlockHeight() < targetHeight {
				continue
			}

			benchDuration := time.Since(t0)
			fmt.Println(float64(benchDuration.Milliseconds()) / float64(targetHeight))

			close(ready)
			return
		}
	}()
}

func (system *CoordinatorBenchSystem) startAndWaitHeightAndKeepRunning(
	ctx context.Context,
	opcfg coordinator.OperationalConfig,
	targetHeight proto.BlockHeight,
	ready chan<- struct{},
	keepRunningUntil <-chan struct{},
) {
	t0 := time.Now()

	system.CoordinatorBenchCluster.StartScheduler(ctx)

	ctl := system.System.Start(ctx, opcfg)
	ctl.SignalGenesisIsReady()

	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, system.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	system.sched.SetCoordScheduler(coordSchedConn)

	system.CoordinatorBenchCluster.StartPump(ctx)

	reachedTarget := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-system.stop:
			return
		default:
		}

		cb := system.notify.ReceiveCommitted(ctx)
		if cb == nil {
			return
		}

		if !reachedTarget && cb.BlockHeight() >= targetHeight {
			reachedTarget = true
			benchDuration := time.Since(t0)
			fmt.Println(float64(benchDuration.Milliseconds()) / float64(targetHeight))
			close(ready)
		}

		if !reachedTarget {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-system.stop:
			return
		case <-keepRunningUntil:
			return
		default:
		}
	}
}
