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
	//
	stopAfterNumCommits *int, // stop after this many commits
	stopAfterHeight *proto.BlockHeight, // stop after this height
	//
	ready chan<- struct{},
	onCommit func(*batonkit.CommittableBlock),
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

	// start submitting txs
	s.CoordinatorBenchCluster.StartPump(ctx)

	// wait for numCommits and signal ready
	go func() {
		numCommitted := 0
		for {
			select {
			case <-ctx.Done():
				liblog.Errorf(ctx, "benchmark loop: context cancelled; exiting (%v)", ctx.Err())
				return
			case <-s.stop:
				liblog.Errorf(ctx, "benchmark loop: terminated; exiting")
				return
			default:
			}

			cb := s.notify.ReceiveCommitted(ctx)
			if cb == nil {
				liblog.Errorf(ctx, "benchmark loop: no committed block received; exiting")
				return
			}
			numCommitted++
			if onCommit != nil {
				onCommit(cb)
			}

			// check stopping conditions
			completed := false
			if stopAfterNumCommits != nil && numCommitted >= *stopAfterNumCommits {
				completed = true
			}
			if stopAfterHeight != nil && cb.BlockHeight() >= *stopAfterHeight {
				completed = true
			}

			if completed {
				// the benchmark is complete, print the results
				benchDuration := time.Since(t0)
				if numCommitted > 0 {
					liblog.Infof(ctx, "coordinator benchmark completed in %s, %f ms/block",
						benchDuration, float64(benchDuration.Milliseconds())/float64(numCommitted))
					// this print is expected by CI metric harness
					fmt.Println(float64(benchDuration.Milliseconds()) / float64(numCommitted))
				} else {
					liblog.Infof(ctx, "coordinator benchmark completed in %s, no blocks committed",
						benchDuration)
				}
				close(ready)
				break
			}
		}

		// keep running for the benefit of other validators
		for {
			select {
			case <-ctx.Done():
				liblog.Errorf(ctx, "benchmark tail: context cancelled; exiting (%v)", ctx.Err())
				return
			case <-s.stop:
				liblog.Errorf(ctx, "benchmark tail: terminated; exiting")
				return
			default:
			}
			if s.notify.ReceiveCommitted(ctx) == nil {
				liblog.Errorf(ctx, "benchmark tail: no committed block received; exiting")
				return
			}
		}
	}()
}
