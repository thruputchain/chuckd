package coordinator_bench

import (
	"context"
	"math/rand"
	"sync"

	"github.com/thruputchain/chuckd/runtime"
	"github.com/thruputchain/chuckd/system/coordinator"
	"github.com/thruputchain/chuckd/test/testnet"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/libtime"
	proto "github.com/thruputchain/thruput-protocol-go"
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
