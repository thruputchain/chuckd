package coordinator_bench

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/net/debug"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type CoordinatorBenchSystem struct {
	inside  libnet.Network
	outside libnet.Network
	notify  *coordinator.NotifyReceive
	*coordinator.System
	*CoordinatorBenchCluster
}

func NewCoordinatorBenchSystem(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	outside libnet.Network,
	cfg proto.Config,
	testCfg CoordinatorBenchConfig,
) *CoordinatorBenchSystem {
	inside := testnet.NewTestNetwork()
	notify := coordinator.NewNotifyReceive()
	s := coordinator.NewSystem(ctx, notify, runtime.NewRandFromRand(rand), cfg, inside, outside)
	return &CoordinatorBenchSystem{
		inside:  inside,
		outside: outside,
		notify:  notify,
		System:  s,
		CoordinatorBenchCluster: NewCoordinatorBenchCluster(
			ctx,
			time,
			runtime.NewRandFromRand(rand),
			inside,
			cfg.Validator_config.Private_info,
			s.WalletSubmitter,
			testCfg,
			false, // use slower testdb with consistency checks
		),
	}
}

func (s *CoordinatorBenchSystem) Start(
	ctx context.Context,
	opcfg coordinator.OperationalConfig,
	numCommits int, // stop after this many commits
	ready chan<- struct{},
) {
	// start debug service and wait until all counterparties are ready
	counterAddr := make([]string, len(s.System.Config().Validator_config.Known_validators))
	colonPort := fmt.Sprintf(":%d", opcfg.DebugPort)
	for i, v := range s.System.Config().Validator_config.Known_validators {
		host, _, err := libnet.StripPort(v.Address)
		must.NoError(ctx, err)
		counterAddr[i] = host + colonPort
	}
	err := debug.StartAndWaitCounterpartiesReady(ctx, s.outside, colonPort, counterAddr)
	must.NoError(ctx, err)
	liblog.Infof("all counterparties are up; starting coordinator benchmark")

	// record the beginning of the benchmark
	t0 := time.Now()

	// start scheduler rpc, needed by coordinator system
	s.CoordinatorBenchCluster.StartScheduler(ctx)

	// start coordinator
	ctl := s.System.Start(ctx, opcfg)
	ctl.GenesisIsReady()

	// connect the test scheduler to the coordinator
	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, s.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	s.sched.SetCoordScheduler(coordSchedConn)

	s.CoordinatorBenchCluster.StartPump(ctx)

	// wait for numCommits and signal ready
	go func() {
		for range numCommits {
			s.notify.ReceiveCommitted(ctx)
		}
		benchDuration := time.Since(t0)
		fmt.Println(
			float64(benchDuration.Milliseconds()) / float64(numCommits), // milliseconds/commit
		)

		// keep consuming commits for 2 more seconds,
		// giving time to stragglers to complete
		go func() {
			for {
				s.notify.ReceiveCommitted(ctx)
			}
		}()
		time.Sleep(time.Second * 2)

		close(ready)
	}()
}

func (s *CoordinatorBenchSystem) Close() {
	s.System.Close()
}
