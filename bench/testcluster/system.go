package testcluster

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sched"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testnet"
	proto "github.com/groundhogchain/protocol-go"
)

type TestClusterSystem struct {
	inside libnet.Network
	*coordinator.System
	*TestCluster
}

func NewTestClusterSystem(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	outside libnet.Network,
	cfg proto.Config,
	testCfg TestClusterConfig,
) *TestClusterSystem {
	inside := testnet.NewTestNetwork()
	s := coordinator.NewSystem(ctx, runtime.NewRandFromRand(rand), cfg, inside, outside)
	return &TestClusterSystem{
		inside: inside,
		System: s,
		TestCluster: NewTestCluster(
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

func (s *TestClusterSystem) Start(ctx context.Context, opcfg coordinator.OperationalConfig) {
	s.TestCluster.Start(ctx, opcfg.InAddr) // start scheduler rpc, needed by coordinator system
	s.System.Start(ctx, opcfg)

	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, s.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	s.sched.SetCoordScheduler(coordSchedConn)
}

func (s *TestClusterSystem) Close() {
	s.System.Close()
}
