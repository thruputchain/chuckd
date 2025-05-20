package testcluster

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/libnet"
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
	rand *rand.Rand,
	cfg proto.Config,
	outside libnet.Network,
) *TestClusterSystem {
	inside := testnet.NewTestNetwork()
	return &TestClusterSystem{
		System:      system.NewSystem(ctx, rand, cfg, inside, outside),
		TestCluster: NewTestCluster(ctx, inside),
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
