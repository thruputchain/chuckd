package aws

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func WriteFiles(
	ctx context.Context,
	dir string,
	cfgs map[string]string,
) {
	os.MkdirAll(dir, 0755)
	for name, cfg := range cfgs {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s.json", name)), []byte(cfg), 0644)
	}
}

func AWSTestClusterConfigs(
	ctx context.Context,
	rand *rand.Rand,
	leaderRegion string,
	quorumRegions []string,
	tps int,
) map[string]string {

	numVs := len(quorumRegions)
	appPort := 4000
	// metricsPort := 2222
	walletPort := 8383

	// compute addresses
	quorumAddrs := make([]string, numVs)
	quorumBinds := make([]string, numVs)
	walletBinds := make([]string, numVs)
	for i := range quorumAddrs {
		quorumAddrs[i] = fmt.Sprintf("chuckd-%s.chuckd.internal:%d", quorumRegions[i], appPort)
		quorumBinds[i] = fmt.Sprintf(":%d", appPort)
		walletBinds[i] = fmt.Sprintf(":%d", walletPort)
	}
	leaderAddr := fmt.Sprintf("chuckd-%s.chuckd.internal:%d", leaderRegion, appPort)
	// leaderBind := fmt.Sprintf(":%d", appPort)

	// gen leader identity
	leaderPriv := common.GenIdentity(ctx, rand, leaderAddr)

	// gen quorum configs
	baseCfg := common.MakeBaseConfigTPS(tps, 250, int64(numVs))
	baseCfg = common.AddSoloLeaderConfig(baseCfg, leaderPriv.Public_info)
	baseCfg.Wallet_config = &proto.WalletConfig{Secret_key: []byte{1, 2, 3}}
	quorumCfgs, qs := common.GenQuorumConfigs(ctx, rand, baseCfg, quorumAddrs)

	// make leader config
	leaderCfg := common.MakeSoloLeaderCfg(ctx, rand, leaderPriv, qs)

	// start deploy script
	cfgs := make(map[string]string)
	cfgs[leaderRegion] = xdrutil.MustXdrToJsonString(&leaderCfg)
	for i, region := range quorumRegions {
		cfgs[region] = xdrutil.MustXdrToJsonString(&quorumCfgs[i])
	}

	return cfgs
}
