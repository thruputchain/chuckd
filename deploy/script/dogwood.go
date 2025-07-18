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
	QUIC bool
	//
	FanOut        int
	BytesPerPay   int
	NumIterations int
	//
	MetricsPort int
	BenchPort   int
	//
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
		suffix[i] = fmt.Sprintf("> bench-dogwood-%d.log 2>&1 &", i)
	}
	scripts := def.ExecScripts(ctx, rand, "chuckd", host, true, suffix)

	return strings.Join(scripts, "\n")
}

func (def BenchDogwoodExecDef) ExecScripts(
	ctx context.Context,
	rand *rand.Rand,
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
		Dogwood_config: &proto.DogwoodConfig{
			Route_count:             7,
			Route_time_to_live_msec: 60e3, // 1 minute
			Route_time_to_use_msec:  200,  // 200 msec
			//
			Endpoint_queue_length:      20,
			Endpoint_max_pending:       400, //XXX: 10x40
			Endpoint_num_clients:       10,
			Endpoint_idle_ttl_msec:     60e3, // 1 minute
			Endpoint_redial_delay_msec: 50,   // 50 msec
			//
			Stream_receiver_queue_length: 20,
			Stream_receiver_worker_count: 40, //XXX
			Stream_sender_worker_count:   40, //XXX
			//
			Transmission_receiver_decode_worker_count: 40,    //XXX
			Transmission_receiver_stream_worker_count: 40000, // streams for ~1GB/sec
			Transmission_sender_worker_count:          40,    //XXX
			//
			Transmission_config: &proto.TransmissionConfig{
				Num_payload_sheaves: 5,
				Num_parity_sheaves:  2,
				Max_chunk_bytes:     60e3, // 60K
				Partial_ttl_msec:    1e3,  // 1 sec
			},
		},
	}

	scripts := make([]string, len(hosts))
	nodeCfgs, _ := common.GenQuorumConfigs(ctx, rand, base, addrs)
	for i, cfg := range nodeCfgs {
		stagger := 0
		if staggerPorts {
			stagger = i
		}
		scripts[i] = benchDogwoodHostScript(ctx, rand, def, i, hosts[i], cfg, chuckdPath, stagger, cmdSuffix[i])
	}
	return scripts
}

func benchDogwoodHostScript(
	ctx context.Context,
	rand *rand.Rand,
	def BenchDogwoodExecDef,
	index int,
	host string,
	cfg proto.Config,
	chuckdPath string,
	stagger int,
	cmdSuffix string,
) string {

	const format = `
#!/bin/sh
set -xe

cat > config-{{ .Stagger }}.json << EOF
{{.ConfigJSON}}
EOF

echo "benchmark working directory $(pwd)"

{{ .ChuckdPath }} -v -l info --config config-{{ .Stagger }}.json \
	{{ if .Def.CPUProfile }}--cpu {{ .CPUProfileName }} {{ end }} \
	{{ if .Def.MemProfile }}--mem {{ .MemProfileName }} {{ end }} \
	{{ if .Def.BlockProfile }}--block {{ .BlockProfileName }} {{ end }} \
	{{ if .Def.TraceProfile }}--trace {{ .TraceProfileName }} {{ end }} \
	--instance node-{{ .NodeIndex }} \
	--seed {{ .Seed }} \
	--metrics :{{ .MetricsPort }} \
	bench dogwood \
	--addr :{{ .BenchPort }} \
	{{ if .Def.QUIC }}--quic{{ end }} \
	--fan-out {{ .Def.FanOut }} \
	--bytes-per-pay {{ .Def.BytesPerPay }} \
	--num-iterations {{ .Def.NumIterations }} \
	{{ .Suffix }}
`
	return textutil.Apply(ctx, format, textutil.M{
		"Def":              def,
		"ConfigJSON":       xdrutil.MustXdrToJsonString(&cfg),
		"NodeIndex":        index,
		"Seed":             rand.Int63(),
		"ChuckdPath":       chuckdPath,
		"MetricsPort":      def.MetricsPort + stagger,
		"BenchPort":        def.BenchPort + stagger,
		"Stagger":          stagger,
		"Suffix":           cmdSuffix,
		"CPUProfileName":   "cpu-" + host + ".prof",
		"MemProfileName":   "mem-" + host + ".prof",
		"BlockProfileName": "block-" + host + ".prof",
		"TraceProfileName": "trace-" + host + ".prof",
	})
}
