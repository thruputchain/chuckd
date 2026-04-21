package coordinator_bench

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libtime"
	proto "github.com/groundhogchain/protocol-go"
)

type CoordinatorBenchSystem struct {
	inside  libnet.Network
	outside libnet.Network
	notify  *coordinator.NotifyReceive
	*coordinator.System
	*CoordinatorBenchCluster
	stopOnce sync.Once
	stop     chan struct{}
}

func NewCoordinatorBenchSystem(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	outside libnet.Network,
	cfg proto.Config,
	testCfg CoordinatorBenchConfig,
	dbDir string,
) *CoordinatorBenchSystem {
	inside := testnet.NewTestNetwork()
	notify := coordinator.NewNotifyReceive()
	s := coordinator.NewSystem(ctx, notify, runtime.NewRandFromRand(rand), cfg, inside, outside)
	return &CoordinatorBenchSystem{
		inside:  inside,
		outside: outside,
		notify:  notify,
		System:  s,
		stop:    make(chan struct{}),
		CoordinatorBenchCluster: NewCoordinatorBenchCluster(
			ctx,
			time,
			runtime.NewRandFromRand(rand),
			inside,
			cfg.Validator_config.Private_info,
			s.UnvettedSubmitter,
			testCfg,
			false, // use slower testdb with consistency checks
			dbDir,
		),
	}
}

func (s *CoordinatorBenchSystem) Close() {
	s.Stop()
}

func (s *CoordinatorBenchSystem) Stop() {
	s.stopOnce.Do(func() {
		close(s.stop)
	})
	s.CoordinatorBenchCluster.Stop()
	s.System.Stop()
}
