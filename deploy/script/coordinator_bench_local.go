package script

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
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
	for _, qcfg := range quorumCfgs {
		untypedVID := protoutil.VIDUntypedString(protoutil.VPublicKeyToID(qcfg.Validator_config.Private_info.Public_info.Public_key))
		fmt.Fprintf(&sh, "cat > config-%s.json << EOF\n", untypedVID)
		fmt.Fprintln(&sh, xdrutil.MustXdrToJsonString(&qcfg))
		fmt.Fprintln(&sh, "EOF")
	}

	// start quorum
	metricsBasePort := 2223
	scpHandleBasePort := 3300
	scpControlBasePort := 4400
	for i, qcfg := range quorumCfgs {
		untypedVID := protoutil.VIDUntypedString(protoutil.VPublicKeyToID(qcfg.Validator_config.Private_info.Public_info.Public_key))
		quorumCmd := fmt.Sprintf(
			`chuckd -l %s --stack --config config-%s.json \
	--instance quorum-%d \
	--seed %d \
	--metrics :%d \
		bench coordinator \
			--tx-per-sec %d \
			--storage-dir %s \
			--out-addr %s \
			--wallet-addr %s \
			--scp-handle-addr %s \
			--scp-control-addr %s \
			--scp-is-validator true`,
			runtime.GetLogLevel(),
			untypedVID,
			i, rand.Int63(), metricsBasePort+i, tps,
			fmt.Sprintf("./storage-%d", i),
			quorumBinds[i], walletBinds[i],
			fmt.Sprintf("127.0.0.1:%d", scpHandleBasePort+i),
			fmt.Sprintf("127.0.0.1:%d", scpControlBasePort+i),
		)
		fmt.Fprintf(&sh, "nohup %s > validator-%s.log 2>&1 &\n", quorumCmd, untypedVID)
		// fmt.Fprintf(&sh, "Q_PID=$! echo $Q_PID > validator-%s.pid\n", untypedVID)
	}

	return sh.String()
}
