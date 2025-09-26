package script

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func BenchTestClusterShellLocal(
	ctx context.Context,
	rand *rand.Rand,
	round time.Duration,
	numVs int,
	fanOut int,
	tps int,
) string {

	quorumPortStart := 50000
	walletPortStart := 40000

	quorumAddrs := make([]string, numVs)
	quorumBinds := make([]string, numVs)
	walletBinds := make([]string, numVs)
	for i := range quorumAddrs {
		quorumAddrs[i] = fmt.Sprintf("127.0.0.1:%d", quorumPortStart+i)
		quorumBinds[i] = fmt.Sprintf(":%d", quorumPortStart+i)
		walletBinds[i] = fmt.Sprintf(":%d", walletPortStart+i)
	}

	// gen quorum configs
	baseCfg := common.MakeBaseConfig(round, int64(numVs), uint32(fanOut))
	baseCfg.Wallet_config = &proto.WalletConfig{Secret_key: []byte{1, 2, 3}}
	quorumCfgs, _ := common.GenQuorumConfigs(ctx, rand, baseCfg, quorumAddrs)

	// start deploy script
	var sh bytes.Buffer
	fmt.Fprintf(&sh, "#!/bin/sh\n")
	fmt.Fprintf(&sh, "set -e\n\n")

	// write quorum configs
	for i, qcfg := range quorumCfgs {
		fmt.Fprintf(&sh, "cat > config-quorum-%d.json << EOF\n", i)
		fmt.Fprintln(&sh, xdrutil.MustXdrToJsonString(&qcfg))
		fmt.Fprintln(&sh, "EOF")
	}

	// start quorum
	metricsBasePort := 2223
	scpHandleBasePort := 3300
	scpControlBasePort := 4400
	for i := range quorumCfgs {
		quorumCmd := fmt.Sprintf(
			`chuckd -l %s --stack --config config-quorum-%d.json \
	--instance quorum-%d \
	--seed %d \
	--metrics :%d \
		bench testcluster \
			--tps %d \
			--out-addr %s \
			--w-addr %s \
			--scp-handle-addr %s \
			--scp-control-addr %s \
			--scp-is-validator true`,
			runtime.GetLogLevel(),
			i, i, rand.Int63(), metricsBasePort+i, tps, quorumBinds[i], walletBinds[i],
			fmt.Sprintf("127.0.0.1:%d", scpHandleBasePort+i),
			fmt.Sprintf("127.0.0.1:%d", scpControlBasePort+i),
		)
		fmt.Fprintf(&sh, "nohup %s > quorum-%d.log 2>&1 &\n", quorumCmd, i)
		// fmt.Fprintf(&sh, "Q_PID=$! echo $Q_PID > quorum-%d.pid\n", i)
	}

	return sh.String()
}
