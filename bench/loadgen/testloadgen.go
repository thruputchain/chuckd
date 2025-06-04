package loadgen

import (
	"context"

	"github.com/groundhogchain/chuckd/bench/testcluster"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/test/testnet"
)

func TestLoadGen(ctx context.Context, tpb int, numDb int, numExec int, callPerExec int) {

	testnet := testnet.NewTestNetwork()

	cluster := testcluster.NewTestCluster(
		ctx,
		libtime.RealTime{},
		runtime.NewRandFromRoot(),
		testnet,
		0,
		nil,
		numDb,
		numExec,
	)

	loadGen := NewLoadGenBench(ctx, runtime.NewRandFromRoot(), testnet, tpb, callPerExec)

	loadGen.Start(ctx, "loadgen")
	cluster.Start(ctx, "loadgen")

	select {}
}
