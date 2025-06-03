package loadgen

import (
	"context"
	"testing"

	"github.com/groundhogchain/chuckd/bench/testcluster"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/test/testnet"
)

func TestLoadGen(t *testing.T) {

	ctx := context.Background()
	liblog.LogVerbosely()
	runtime.SetRootSeed(1)

	testnet := testnet.NewTestNetwork()

	cluster := testcluster.NewTestCluster(
		ctx,
		libtime.RealTime{},
		runtime.NewRandFromRoot(),
		testnet,
		0,
		nil,
	)

	loadGen := NewLoadGenBench(ctx, runtime.NewRandFromRoot(), testnet, 100)

	loadGen.Start(ctx, "loadgen")
	cluster.Start(ctx, "loadgen")

	select {}
}
