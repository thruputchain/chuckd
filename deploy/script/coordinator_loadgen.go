package script

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type CoordinatorLoadgenConfig struct {
	ValidatorPort  int // 50000
	WalletPort     int // 40000
	MetricsPort    int // 2223
	DebugPort      int // 5500
	ScpHandlePort  int // 3300
	ScpControlPort int // 4400
}

func CoordinatorLoadgenScripts(
	// coordinator
	ctx context.Context,
	rand *rand.Rand,
	bcfg CoordinatorLoadgenConfig,
	hosts []string,
	round time.Duration,
	fanOut int,
	// loadgen
	numAccts int,
	txsPerBlock int,
	numBlocks int,
	loadgenAddr string,
	schedAddr string,
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
		scripts[i] = CoordinatorLoadgenScript(
			ctx,
			rand,
			bcfg,
			round,
			fanOut,
			numAccts,
			txsPerBlock,
			numBlocks,
			vcfg,
			hosts[i],
			vaddrs[i],
			fmt.Sprintf(":%d", bcfg.ValidatorPort),
			fmt.Sprintf(":%d", bcfg.WalletPort),
			loadgenAddr,
			schedAddr,
		)
	}

	return scripts
}

func CoordinatorLoadgenScript(
	ctx context.Context,
	rand *rand.Rand,
	bcfg CoordinatorLoadgenConfig,
	// experiment params
	round time.Duration,
	fanOut int,
	//
	numAccts int,
	txsPerBlock int,
	numBlocks int,
	// node params
	vcfg proto.Config,
	host string,
	vAddr string,
	vBind string,
	wBind string,
	loadgenAddr string,
	schedAddr string,
) string {

	var script bytes.Buffer
	fmt.Fprintf(&script, "#!/bin/sh\n")
	fmt.Fprintf(&script, "set -e\n")

	fmt.Fprintf(&script, `
if [ $# -ne 2 ]; then
  echo "Usage: $0 <chuckd_path> <scpd_path>"
  exit 1
fi
`)

	untypedVID := protoutil.VIDUntypedString(
		protoutil.VPublicKeyToID(
			vcfg.Validator_config.Private_info.Public_info.Public_key,
		),
	)

	// build config file
	fmt.Fprintf(&script, "cat > config-%s.json << EOF\n", host)
	fmt.Fprintln(&script, libxdr.MustXdrToJsonString(&vcfg))
	fmt.Fprintln(&script, "EOF")

	storageDir := fmt.Sprintf("./storage-%s", host)

	startCmd := fmt.Sprintf(
		`$1 -l %s --stack --config config-%s.json \
	--instance %s \
	--seed %d \
	--metrics :%d \
		bench coordinator_loadgen \
			--debug-port %d \
			--num-accts %d \
			--txs-per-block %d \
			--num-blocks %d \
			--storage-dir %s \
			--out-addr %s \
			--wallet-addr %s \
			--loadgen-addr %s \
			--sched-addr %s \
			--scpd-path $2 \
			--scp-handle-addr %s \
			--scp-control-addr %s \
			--scp-is-validator true`,
		liblog.GetLogLevel().String(),
		host,
		untypedVID,
		rand.Int63(),
		bcfg.MetricsPort,
		bcfg.DebugPort,
		numAccts,
		txsPerBlock,
		numBlocks,
		storageDir,
		vBind,
		wBind,
		loadgenAddr,
		schedAddr,
		fmt.Sprintf("127.0.0.1:%d", bcfg.ScpHandlePort),
		fmt.Sprintf("127.0.0.1:%d", bcfg.ScpControlPort),
	)

	fmt.Fprintf(&script, "rm -rf %s\n", storageDir)
	fmt.Fprintf(&script, `%s \
				> >(tee validator-%s.report) \
				2> >(tee validator-%s.log >&2)
`, startCmd, host, host)

	return script.String()
}
