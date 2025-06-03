package testcluster

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system"
	"github.com/groundhogchain/chuckd/test/testnet"
	proto "github.com/groundhogchain/protocol-go"
)

type TestClusterSystem struct {
	*system.System
	*TestCluster
}

func NewTestClusterSystem(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	cfg proto.Config,
	outside libnet.Network,
	tps int64,
) *TestClusterSystem {
	inside := testnet.NewTestNetwork()
	sys := system.NewSystem(ctx, runtime.NewRandFromRand(rand), cfg, inside, outside)
	return &TestClusterSystem{
		System:      sys,
		TestCluster: NewTestCluster(ctx, time, runtime.NewRandFromRand(rand), inside, tps, sys.Submitter, 5, 7),
	}
}

func (sys *TestClusterSystem) Start(ctx context.Context, outAddr string, walletAddr string) {
	inAddr := coordAddress
	sys.System.Start(ctx, inAddr, outAddr, walletAddr)
	sys.TestCluster.Start(ctx, inAddr)
}

func (sys *TestClusterSystem) Close() {
	sys.System.Close()
}
