package k8s

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/groundhogchain/chuckd/deploy/common"
	proto "github.com/groundhogchain/protocol-go"
)

func BenchBroadcastBlueprint(ctx context.Context, rand *rand.Rand, numVs int) string {

	// make services for nodes
	serviceName := "chuckd-bench-broadcast"
	services, addrs := MakeServices(serviceName, numVs)

	// gen quorum configs
	base := common.MakeBaseConfigTPS(1e5, 250, int64(numVs))
	cfgs, _ := common.GenQuorumConfigs(ctx, rand, base, addrs)

	// make quorum statefulset
	rs := MakeStatefulSet(
		serviceName,
		"groundhogchain:chuckd.latest",
		"bench broadcast --addr :9111 --fanout 5 --iterations 3 --pay-per-chan 10 --paysize 100",
		cfgs,
		services,
	)

	return SprintResources(ctx, rs)
}

func BenchTestClusterBlueprint(ctx context.Context, rand *rand.Rand, numVs int, tps int) string {

	// make services for validator nodes
	serviceNameQuorum := "chuckd-bench-testcluster-quorum"
	quorumServices, quorumAddrs := MakeServices(serviceNameQuorum, numVs)

	// make service for scp solo leader
	serviceNameLeader := "chuckd-bench-testcluster-leader"
	leaderServices, leaderAddrs := MakeServices(serviceNameLeader, 1)

	// gen leader identity
	leaderPriv := common.GenIdentity(ctx, rand, leaderAddrs[0])

	// gen quorum configs
	baseCfg := common.MakeBaseConfigTPS(tps, 250, int64(numVs))
	baseCfg = common.AddSoloLeaderConfig(baseCfg, leaderPriv.Public_info)
	baseCfg.Wallet_config = &proto.WalletConfig{Secret_key: []byte{1, 2, 3}}
	quorumCfgs, qs := common.GenQuorumConfigs(ctx, rand, baseCfg, quorumAddrs)

	// make leader config
	leaderCfg := common.MakeSoloLeaderCfg(ctx, rand, leaderPriv, qs)

	// make quorum statefulset
	quorumResources := MakeStatefulSet(
		serviceNameQuorum,
		"groundhogchain:chuckd.latest",
		fmt.Sprintf("bench testcluster --tps %d --out-addr :9111 --w-addr :6666", tps),
		quorumCfgs,
		quorumServices,
	)

	// stateful set for solo leader
	leaderResources := MakeStatefulSet(
		serviceNameLeader,
		"groundhogchain:chuckd.latest",
		fmt.Sprintf("solo leader --addr :9111 --delay %d", 100), // slot delay 1/10 sec
		[]proto.Config{leaderCfg},
		leaderServices,
	)

	return SprintResources(ctx, append(leaderResources, quorumResources...))
}
