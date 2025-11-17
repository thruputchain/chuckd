package coordinator_bench

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testnet"
	proto "github.com/groundhogchain/protocol-go"
)

type CoordinatorBenchSystem struct {
	inside libnet.Network
	notify *coordinator.NotifyReceive
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
		inside: inside,
		notify: notify,
		System: s,
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
	s.CoordinatorBenchCluster.Start(ctx, opcfg.InAddr) // start scheduler rpc, needed by coordinator system
	s.System.Start(ctx, opcfg)

	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, s.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	s.sched.SetCoordScheduler(coordSchedConn)

	// wait for numCommits and signal ready
	go func() {
		for i := 0; i < numCommits; i++ {
			s.notify.ReceiveCommitted(ctx)
		}
		// XXX: print duration from start of experiment after trigger
		close(ready)
	}()
}

func (s *CoordinatorBenchSystem) Close() {
	s.System.Close()
}
