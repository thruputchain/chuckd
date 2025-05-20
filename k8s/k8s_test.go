package k8s

import (
	"context"
	"fmt"
	"testing"

	"github.com/groundhogchain/chuckd/runtime"
)

func TestBenchBroadcast(t *testing.T) {

	runtime.SetRootSeed(0)
	ctx := context.Background()

	serviceName := "chuckd-bench-broadcast"
	numVs := 3
	base := MakeBaseConfigTPS(1e5, 250, int64(numVs))
	services, addrs := MakeServices(serviceName, numVs)
	cfgs := GenNetworkConfigs(ctx, runtime.NewRandFromRoot(), base, addrs)
	rs := MakeNetworkStatefulSet(serviceName, "groundhogchain:chuckd.latest", BenchBroadcastSubcmd, cfgs, services)
	fmt.Println(SprintResources(ctx, rs))
}
