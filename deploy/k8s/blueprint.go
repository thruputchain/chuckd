package k8s

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/deploy/common"
	proto "github.com/groundhogchain/protocol-go"
)

func BenchDogwoodBlueprint(ctx context.Context, rand *rand.Rand, numVs int, fanOut int) string {

	// make services for nodes
	serviceName := "chuckd-bench-broadcast"
	services, addrs := MakeServices(serviceName, numVs)

	// gen quorum configs
	base := common.MakeBaseConfig(time.Second, int64(numVs), uint32(fanOut))
	cfgs, _ := common.GenQuorumConfigs(ctx, rand, base, addrs)

	// make quorum statefulset
	rs := MakeStatefulSet(
		serviceName,
		"groundhogchain:chuckd.latest",
		"bench dogwood --addr :9111 --num-iterations 3 --bytes-per-pay 300000",
		cfgs,
		services,
	)

	return SprintResources(ctx, rs)
}

func BenchTestClusterBlueprint(ctx context.Context, rand *rand.Rand, round time.Duration, numVs int, fanOut int, tps int) string {

	// make services for validator nodes
	serviceNameQuorum := "chuckd-bench-testcluster-quorum"
	quorumServices, quorumAddrs := MakeServices(serviceNameQuorum, numVs)

	// gen quorum configs
	baseCfg := common.MakeBaseConfig(round, int64(numVs), uint32(fanOut))
	baseCfg.Wallet_config = &proto.WalletConfig{Secret_key: []byte{1, 2, 3}}
	quorumCfgs, _ := common.GenQuorumConfigs(ctx, rand, baseCfg, quorumAddrs)

	// make quorum statefulset
	quorumResources := MakeStatefulSet(
		serviceNameQuorum,
		"groundhogchain:chuckd.latest",
		//XXX: missing scpd flags
		fmt.Sprintf("bench testcluster --tps %d --out-addr :9111 --w-addr :6666", tps),
		quorumCfgs,
		quorumServices,
	)

	return SprintResources(ctx, quorumResources)
}
