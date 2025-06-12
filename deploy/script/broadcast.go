package script

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/groundhogchain/chuckd/deploy/common"
	"github.com/groundhogchain/chuckd/util/textutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type BenchBroadcastDef struct {
	QUIC bool
	//
	FanOut          int
	ChunkPerChannel int
	BytesPerSec     int
	NumIterations   int
	//
	ChunkSize        int
	MulticastTreeTTL time.Duration
	//
	MetricsPort int
	BenchPort   int
}

func (def BenchBroadcastDef) LocalScript(
	ctx context.Context,
	rand *rand.Rand,
	chuckdPath string,
	numHost int,
) string {

	host := make([]string, numHost)
	suffix := make([]string, numHost)
	for i := range host {
		host[i] = "localhost"
		suffix[i] = fmt.Sprintf("> bb-%d.log 2>&1 &", i)
	}
	scripts := def.ExecScripts(ctx, rand, "chuckd", host, true, suffix)

	return strings.Join(scripts, "\n")
}

func (def BenchBroadcastDef) ExecScripts(
	ctx context.Context,
	rand *rand.Rand,
	chuckdPath string,
	hosts []string,
	staggerPorts bool,
	cmdSuffix []string,
) (perHost []string) {

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
		Broadcast_config: &proto.BroadcastConfig{
			Multicast_tree_ttl_msec: def.MulticastTreeTTL.Milliseconds(), // how long to cache multicast tree topology
			Chunk_size:              uint64(def.ChunkSize),               // size of individual broadcast chunks
		},
	}

	scripts := make([]string, len(hosts))
	nodeCfgs, _ := common.GenQuorumConfigs(ctx, rand, base, addrs)
	for i, cfg := range nodeCfgs {
		stagger := 0
		if staggerPorts {
			stagger = i
		}
		scripts[i] = benchBroadcastHostScript(ctx, rand, def, i, cfg, chuckdPath, stagger, cmdSuffix[i])
	}
	return scripts
}

func benchBroadcastHostScript(
	ctx context.Context,
	rand *rand.Rand,
	def BenchBroadcastDef,
	index int,
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

{{ .ChuckdPath }} -v -l info --config config.json-{{ .Stagger }} \
	--instance node-{{ .NodeIndex }} \
	--seed {{ .Seed }} \
	--metrics :{{ .MetricsPort }} \
	bench broadcast \
	--addr :{{ .BenchPort }} \
	{{ if .Def.QUIC }}--quic{{ end }} \
	--fan-out {{ .Def.FanOut }} \
	--chunk-per-chan {{ .Def.ChunkPerChannel }} \
	--bytes-per-sec {{ .Def.BytesPerSec }} \
	--num-iterations {{ .Def.NumIterations }} \
	{{ .Suffix }}
`
	return textutil.Apply(ctx, format, textutil.M{
		"Def":         def,
		"ConfigJSON":  xdrutil.MustXdrToJsonString(&cfg),
		"NodeIndex":   index,
		"Seed":        rand.Int63(),
		"ChuckdPath":  chuckdPath,
		"MetricsPort": def.MetricsPort + stagger,
		"BenchPort":   def.BenchPort + stagger,
		"Stagger":     stagger,
		"Suffix":      cmdSuffix,
	})
}
