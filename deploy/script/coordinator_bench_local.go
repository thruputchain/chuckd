package script

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/thruputchain/chuckd/util/configtext"
	"github.com/thruputchain/chuckd/deploy/common"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/liblog"
)

func CoordinatorBenchShellLocal(
	ctx context.Context,
	rand *rand.Rand,
	round time.Duration,
	numVs int,
	fanOut int,
	tps int,
) string {

	validatorPortStart := 50000
	walletPortStart := 40000

	validatorAddrs := make([]string, numVs)
	validatorBinds := make([]string, numVs)
	walletBinds := make([]string, numVs)
	for i := range validatorAddrs {
		validatorAddrs[i] = fmt.Sprintf("127.0.0.1:%d", validatorPortStart+i)
		validatorBinds[i] = fmt.Sprintf(":%d", validatorPortStart+i)
		walletBinds[i] = fmt.Sprintf(":%d", walletPortStart+i)
	}

	// gen quorum configs
	baseCfg := common.MakeBaseConfig(round, int64(numVs), uint32(fanOut))
	quorumCfgs, _ := common.GenQuorumConfigs(ctx, rand, baseCfg, validatorAddrs)

	// start deploy script
	var sh bytes.Buffer
	fmt.Fprintf(&sh, "#!/bin/sh\n")
	fmt.Fprintf(&sh, "set -e\n\n")

	// write quorum configs
	for _, qcfg := range quorumCfgs {
		untypedVID := protoutil.VIDUntypedString(
			protoutil.VPublicKeyToID(
				qcfg.Validator_config.Private_info.Public_info.Public_key))
		fmt.Fprintf(&sh, "cat > config-%s.json << EOF\n", untypedVID)
		fmt.Fprintln(&sh, configtext.MustMarshalString(&qcfg))
		fmt.Fprintln(&sh, "EOF")
	}

	// start quorum
	metricsBasePort := 2223
	debugBasePort := 5500
	scpHandleBasePort := 3300
	scpControlBasePort := 4400
	for i, qcfg := range quorumCfgs {
		untypedVID := protoutil.VIDUntypedString(
			protoutil.VPublicKeyToID(
				qcfg.Validator_config.Private_info.Public_info.Public_key))
		validatorCmd := fmt.Sprintf(
			`chuckd -l %s --stack --config config-%s.json \
	--instance quorum-%d \
	--seed %d \
	--metrics :%d \
		bench coordinator \
			--debug-port %d \
			--txs-per-sec %d \
			--storage-dir %s \
			--out-addr %s \
			--wallet-addr %s \
			--scp-handle-addr %s \
			--scp-control-addr %s \
			--scp-is-validator true`,
			liblog.GetLogLevel().String(),
			untypedVID,
			i,
			rand.Int63(),
			metricsBasePort+i,
			debugBasePort+i,
			tps,
			fmt.Sprintf("./storage-%d", i),
			validatorBinds[i],
			walletBinds[i],
			fmt.Sprintf("127.0.0.1:%d", scpHandleBasePort+i),
			fmt.Sprintf("127.0.0.1:%d", scpControlBasePort+i),
		)
		fmt.Fprintf(&sh, "rm -rf %s\n", fmt.Sprintf("./storage-%d", i))
		fmt.Fprintf(&sh, "nohup %s > validator-%s.log 2>&1 &\n", validatorCmd, untypedVID)
		// fmt.Fprintf(&sh, "Q_PID=$! echo $Q_PID > validator-%s.pid\n", untypedVID)
	}

	return sh.String()
}
