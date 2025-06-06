package metal

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func BenchTestClusterShell(
	ctx context.Context,
	rand *rand.Rand,
	numVs int,
	tps int,
) string {

	quorumPortStart := 50000
	walletPortStart := 40000
	leaderPort := 60000
	leaderDelayMsec := 100

	quorumAddrs := make([]string, numVs)
	quorumBinds := make([]string, numVs)
	walletBinds := make([]string, numVs)
	for i := range quorumAddrs {
		quorumAddrs[i] = fmt.Sprintf("localhost:%d", quorumPortStart+i)
		quorumBinds[i] = fmt.Sprintf(":%d", quorumPortStart+i)
		walletBinds[i] = fmt.Sprintf(":%d", walletPortStart+i)
	}
	leaderAddr := fmt.Sprintf("localhost:%d", leaderPort)
	leaderBind := fmt.Sprintf(":%d", leaderPort)

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
	var sh bytes.Buffer
	fmt.Fprintf(&sh, "#!/bin/sh\n")
	fmt.Fprintf(&sh, "set -e\n\n")

	// write leader config
	fmt.Fprintf(&sh, "cat > config-leader.json << EOF\n")
	fmt.Fprintln(&sh, xdrutil.MustXdrToJsonString(&leaderCfg))
	fmt.Fprintln(&sh, "EOF")

	// write quorum configs
	for i, qcfg := range quorumCfgs {
		fmt.Fprintf(&sh, "cat > config-quorum-%d.json << EOF\n", i)
		fmt.Fprintln(&sh, xdrutil.MustXdrToJsonString(&qcfg))
		fmt.Fprintln(&sh, "EOF")
	}

	// start leader
	leaderCmd := fmt.Sprintf(
		"chuckd -v --config config-leader.json --instance %s --seed %d --metrics :2222 "+
			"solo leader --addr %s --delay %d",
		"leader", rand.Int63(), leaderBind, leaderDelayMsec,
	)
	fmt.Fprintf(&sh, "nohup %s > leader.log 2>&1 &\n", leaderCmd)
	// fmt.Fprintf(&sh, "L_PID=$! echo $L_PID > leader.pid\n")

	// start quorum
	metricsBasePort := 2223
	for i := range quorumCfgs {
		quorumCmd := fmt.Sprintf(
			"chuckd -v --config config-quorum-%d.json --instance quorum-%d --seed %d --metrics :%d "+
				"bench testcluster --tps %d --out-addr %s --w-addr %s",
			i, i, rand.Int63(), metricsBasePort+i, tps, quorumBinds[i], walletBinds[i],
		)
		fmt.Fprintf(&sh, "nohup %s > quorum-%d.log 2>&1 &\n", quorumCmd, i)
		// fmt.Fprintf(&sh, "Q_PID=$! echo $Q_PID > quorum-%d.pid\n", i)
	}

	return sh.String()
}
