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

type CoordinatorBenchConfig struct {
	ChuckdPath     string
	ScpdPath       string
	ValidatorPort  int // 50000
	WalletPort     int // 40000
	MetricsPort    int // 2223
	DebugPort      int // 5500
	ScpHandlePort  int // 3300
	ScpControlPort int // 4400
}

func CoordinatorBenchScripts(
	ctx context.Context,
	rand *rand.Rand,
	bcfg CoordinatorBenchConfig,
	hosts []string,
	round time.Duration,
	fanOut int,
	txsPerSec int,
	numCommits int,
) []string {

	nv := len(hosts)

	vaddrs := make([]string, nv)
	for i, host := range hosts {
		vaddrs[i] = fmt.Sprintf("%s:%d", host, bcfg.ValidatorPort)
	}

	baseCfg := common.MakeBaseConfig(round, int64(nv), uint32(fanOut))
	baseCfg.Wallet_config = &proto.WalletConfig{Secret_key: []byte{1, 2, 3}}
	vcfgs, _ := common.GenQuorumConfigs(ctx, rand, baseCfg, vaddrs)

	scripts := make([]string, nv)
	for i, vcfg := range vcfgs {
		scripts[i] = CoordinatorBenchScript(
			ctx,
			rand,
			bcfg,
			round,
			fanOut,
			txsPerSec,
			numCommits,
			vcfg,
			hosts[i],
			vaddrs[i],
			fmt.Sprintf(":%d", bcfg.ValidatorPort),
			fmt.Sprintf(":%d", bcfg.WalletPort),
		)
	}

	return scripts
}

func CoordinatorBenchScript(
	ctx context.Context,
	rand *rand.Rand,
	bcfg CoordinatorBenchConfig,
	// experiment params
	round time.Duration,
	fanOut int,
	txsPerSec int,
	numCommits int,
	// node params
	vcfg proto.Config,
	host string,
	vAddr string,
	vBind string,
	wBind string,
) string {

	var script bytes.Buffer
	fmt.Fprintf(&script, "#!/bin/sh\n")
	fmt.Fprintf(&script, "set -e\n\n")

	untypedVID := protoutil.VIDUntypedString(
		protoutil.VPublicKeyToID(
			vcfg.Validator_config.Private_info.Public_info.Public_key,
		),
	)

	// build config file
	fmt.Fprintf(&script, "cat > config-%s.json << EOF\n", host)
	fmt.Fprintln(&script, xdrutil.MustXdrToJsonString(&vcfg))
	fmt.Fprintln(&script, "EOF")

	storageDir := fmt.Sprintf("./validator-storage-%s", host)

	startCmd := fmt.Sprintf(
		`%s -l %s --stack --config config-%s.json \
	--instance %s \
	--seed %d \
	--metrics :%d \
		bench coordinator \
			--debug-port %d \
			--tx-per-sec %d \
			--num-commits %d \
			--storage-dir %s \
			--out-addr %s \
			--wallet-addr %s \
			--scp-handle-addr %s \
			--scp-control-addr %s \
			--scp-is-validator true`,
		bcfg.ChuckdPath,
		runtime.GetLogLevel(),
		host,
		untypedVID,
		rand.Int63(),
		bcfg.MetricsPort,
		bcfg.DebugPort,
		txsPerSec,
		numCommits,
		storageDir,
		vBind,
		wBind,
		fmt.Sprintf("127.0.0.1:%d", bcfg.ScpHandlePort),
		fmt.Sprintf("127.0.0.1:%d", bcfg.ScpControlPort),
	)

	fmt.Fprintf(&script, "rm -rf %s\n", storageDir)
	fmt.Fprintf(&script, "%s > validator-%s.log 2>&1 &\n", startCmd, host)

	return script.String()
}
