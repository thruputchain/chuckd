package testcluster

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testnet"
	proto "github.com/groundhogchain/protocol-go"
)

type TestClusterSystem struct {
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
	callsPerExec int,
	batchSize int,
) *TestClusterSystem {
	inside := testnet.NewTestNetwork()
	sys := coordinator.NewSystem(ctx, runtime.NewRandFromRand(rand), cfg, inside, outside, callsPerExec, batchSize)
	return &TestClusterSystem{
		System: sys,
		TestCluster: NewTestCluster(
			ctx,
			time,
			runtime.NewRandFromRand(rand),
			inside,
			sys.Submitter,
			testCfg,
			false, // use slower testdb with consistency checks
		),
	}
}

func (sys *TestClusterSystem) Start(ctx context.Context, inAddr string, outAddr string, walletAddr string) {
	sys.System.Start(ctx, inAddr, outAddr, walletAddr)
	sys.TestCluster.Start(ctx, inAddr)
}

func (sys *TestClusterSystem) Close() {
	sys.System.Close()
}
