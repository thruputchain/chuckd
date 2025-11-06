package script

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/util/textutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type BenchDogwoodExecDef struct {
	TreeFanOut    uint32
	BytesPerPay   int
	NumIterations int
	//
	MetricsPort int
	BenchPort   int
	//
	Stack        bool
	CPUProfile   bool
	MemProfile   bool
	BlockProfile bool
	TraceProfile bool
}

func (def BenchDogwoodExecDef) LocalScript(
	ctx context.Context,
	rand *rand.Rand,
	chuckdPath string,
	numHost int,
) string {

	host := make([]string, numHost)
	suffix := make([]string, numHost)
	for i := range host {
		host[i] = "localhost"
		suffix[i] = fmt.Sprintf("> dogwood-bench-%d.log 2>&1 &", i)
	}
	scripts := def.ExecScripts(ctx, rand, "", "chuckd", host, true, suffix)

	return strings.Join(scripts, "\n")
}

func (def BenchDogwoodExecDef) HostScripts(
	ctx context.Context,
	rand *rand.Rand,
	chuckdPath string,
	host ...string,
) []string {

	suffix := make([]string, len(host))
	id := make([]string, len(host))
	for i := range host {
		suffix[i] = ""
		id[i] = fmt.Sprintf("host=%s-node=%d", host[i], i)
	}
	preamble := `
if [ $# -ne 1 ]; then
  echo "Usage: $0 <chuckd_path>"
  exit 1
fi
`
	return def.ExecScripts(ctx, rand, preamble, "$1", host, false, suffix)
}

func (def BenchDogwoodExecDef) ExecScripts(
	ctx context.Context,
	rand *rand.Rand,
	preamble string, // script preamble
	chuckdPath string,
	hosts []string,
	staggerPorts bool,
	cmdSuffix []string,
) (perHost []string) {

	if len(cmdSuffix) == 0 {
		cmdSuffix = make([]string, len(hosts))
	}

	addrs := make([]string, len(hosts))
	for i, host := range hosts {
		stagger := 0
		if staggerPorts {
			stagger = i
		}
		addrs[i] = fmt.Sprintf("%s:%d", host, def.BenchPort+stagger)
	}

	base := proto.Config{
		Validator_config: nil,
		Router_config: &proto.RouterConfig{
			Tree_fan_out: def.TreeFanOut,
		},
		Scp_config: &proto.SCPConfig{},
		Dogwood_config: &proto.DogwoodConfig{
			Route_count:             17,
			Route_time_to_live_msec: 60e3, // 1 minute
			Route_max_latency_msec:  1e3,  // 1 sec
			//
			Endpoint_max_pending: 16 * 50,
			Endpoint_num_clients: 16,
			//
			Endpoint_idle_ttl_msec:     60e3, // 1 minute
			Endpoint_dial_timeout_msec: 2e3,  // 2 sec
			Endpoint_redial_delay_msec: 100,  // 100 msec
			Endpoint_call_timeout_msec: 2e3,  // 2 sec
			//
			Stream_receiver_worker_count: 512,
			Stream_sender_worker_count:   32,
			//
			Transmission_receiver_decode_worker_count: 512,
			Transmission_sender_worker_count:          32,
			//
			Transmission_partial_max_ttl_msec:  10e3, // 10 sec
			Transmission_partial_idle_ttl_msec: 1e3,  // 1 sec
		},
	}

	scripts := make([]string, len(hosts))
	nodeCfgs, _ := common.GenQuorumConfigs(ctx, rand, base, addrs)
	for i, cfg := range nodeCfgs {
		stagger := 0
		if staggerPorts {
			stagger = i
		}
		scripts[i] = benchDogwoodHostScript(
			ctx,
			rand,
			def,
			preamble,
			i,
			hosts[i],
			cfg,
			chuckdPath,
			stagger,
			cmdSuffix[i],
		)
	}
	return scripts
}

func benchDogwoodHostScript(
	ctx context.Context,
	rand *rand.Rand,
	def BenchDogwoodExecDef,
	preamble string,
	index int,
	host string,
	cfg proto.Config,
	chuckdPath string,
	stagger int,
	cmdSuffix string,
) string {

	const format = `
#!/bin/sh
# chmod +x dogwood-bench-{{ .Name }}.sh

set -xe

{{ .Preamble }}

cat > {{ .ConfigName }} << EOF
{{.ConfigJSON}}
EOF

{{ .Prefix }} \
	{{ .ChuckdPath }} \
	-l {{ .LogLevel }} \
	--config {{ .ConfigName }} \
	{{ if .Def.Stack }}--stack {{ end }} \
	{{ if .Def.CPUProfile }}--cpu {{ .CPUProfileName }} {{ end }} \
	{{ if .Def.MemProfile }}--mem {{ .MemProfileName }} {{ end }} \
	{{ if .Def.BlockProfile }}--block {{ .BlockProfileName }} {{ end }} \
	{{ if .Def.TraceProfile }}--trace {{ .TraceProfileName }} {{ end }} \
	--instance node-{{ .NodeIndex }} \
	--seed {{ .Seed }} \
	--metrics :{{ .MetricsPort }} \
	bench dogwood \
	--addr :{{ .BenchPort }} \
	--bytes-per-pay {{ .Def.BytesPerPay }} \
	--num-iterations {{ .Def.NumIterations }} \
	{{ .Suffix }} 2> {{ .LogName }} | tee {{ .ReportName }}
`

	name := fmt.Sprintf("host=%s,seq=%d", host, index)
	return textutil.Apply(ctx, format, textutil.M{
		"Preamble":         preamble,
		"LogLevel":         "info",
		"Prefix":           "perf stat -e cache-misses,cache-references,L1-dcache-loads,L1-dcache-load-misses,L1-icache-load-misses,LLC-loads,LLC-load-misses,branch-misses,page-faults",
		"Def":              def,
		"Name":             name,
		"Host":             host,
		"ConfigName":       "dogwood-bench-config-" + name + ".json",
		"ConfigJSON":       xdrutil.MustXdrToJsonString(&cfg),
		"NodeIndex":        index,
		"Seed":             rand.Int63(),
		"ChuckdPath":       chuckdPath,
		"MetricsPort":      def.MetricsPort + stagger,
		"BenchPort":        def.BenchPort + stagger,
		"Stagger":          stagger,
		"Suffix":           cmdSuffix,
		"CPUProfileName":   "dogwood-bench-cpu-" + name + ".prof",
		"MemProfileName":   "dogwood-bench-mem-" + name + ".prof",
		"BlockProfileName": "dogwood-bench-block-" + name + ".prof",
		"TraceProfileName": "dogwood-bench-trace-" + name + ".prof",
		"LogName":          "dogwood-bench-" + name + ".log",
		"ReportName":       "dogwood-bench-" + name + ".report",
	})
}
