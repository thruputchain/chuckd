package script

import (
	"context"
	"fmt"
	"math/rand"
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
	NumIterations   int
	//
	ChunkSize        int
	MulticastTreeTTL time.Duration
	//
	Port int
}

func (def BenchBroadcastDef) ExecScripts(
	ctx context.Context,
	rand *rand.Rand,
	hosts []string,
) []string {

	addrs := make([]string, len(hosts))
	for i, host := range hosts {
		addrs[i] = fmt.Sprintf("%s:%d", host, def.Port)
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
		scripts[i] = benchBroadcastHostScript(ctx, rand, def, i, cfg)
	}
	return scripts
}

func benchBroadcastHostScript(
	ctx context.Context,
	rand *rand.Rand,
	def BenchBroadcastDef,
	index int,
	cfg proto.Config,
) string {

	const format = `
#!/bin/sh
set -xe

cat > config.json << EOF
{{.ConfigJSON}}
EOF

./chuckd -v --config config.json \
	--instance node-{{ .NodeIndex }} \
	--seed {{ .Seed }} \
	--metrics :2222 \
	bench broadcast \
	--addr :{{ .Def.Port }} \
	{{ if .Def.QUIC }}--quic{{ end }} \
	--fan-out {{ .Def.FanOut }} \
	--chunk-per-chan {{ .Def.ChunkPerChan }} \
	--num-iterations {{ .Def.NumIterations }}
`
	return textutil.Apply(ctx, format, textutil.M{
		"Def":        def,
		"ConfigJSON": xdrutil.MustXdrToJsonString(&cfg),
		"NodeIndex":  index,
		"Seed":       rand.Int63(),
	})
}
