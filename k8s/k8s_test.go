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

	base := MakeBaseConfigTPS(1e5, 250, 3)
	cfgs := GenNetworkConfigs(ctx, runtime.NewRandFromRoot(), base, []string{"X1", "X2", "X3"})
	rs := MakeNetworkStatefulSet("chuckd-bench", "groundhogchain:chuckd.latest", BenchBroadcastSubcmd, cfgs)
	fmt.Println(SprintResources(ctx, rs))
}
