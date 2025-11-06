package script

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/textutil"
)

type BenchEchoExecDef struct {
	NumConns        int
	MaxPendingCalls int
	NumIterations   int
	BytesPerPay     int
	NoEcho          bool
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

func (def BenchEchoExecDef) ExecScripts(
	ctx context.Context,
	rand *rand.Rand,
	chuckdPath string,
	hosts []string,
	cmdSuffix string,
) (perHost []string) {

	must.Assert(ctx, len(hosts) == 2)

	scripts := make([]string, len(hosts))
	for i := range hosts {
		scripts[i] = benchEchoHostScript(ctx, rand, def, i, hosts[i], hosts[(i+1)%2], chuckdPath, cmdSuffix, i == 0)
	}

	return scripts
}

func benchEchoHostScript(
	ctx context.Context,
	rand *rand.Rand,
	def BenchEchoExecDef,
	index int,
	ourHost string,
	theirHost string,
	chuckdPath string,
	cmdSuffix string,
	doSend bool,
) string {

	const format = `
#!/bin/sh
set -xe

{{ .Prefix }} {{ .ChuckdPath }} -l {{ .LogLevel }} \
	{{ if .Def.Stack }}--stack {{ end }} \
	{{ if .Def.CPUProfile }}--cpu {{ .CPUProfileName }} {{ end }} \
	{{ if .Def.MemProfile }}--mem {{ .MemProfileName }} {{ end }} \
	{{ if .Def.BlockProfile }}--block {{ .BlockProfileName }} {{ end }} \
	{{ if .Def.TraceProfile }}--trace {{ .TraceProfileName }} {{ end }} \
	--instance=node-{{ .NodeIndex }} \
	--seed={{ .Seed }} \
	--metrics=:{{ .MetricsPort }} \
	bench echo \
	--do-send={{ .DoSend }} \
	--our-addr=:{{ .BenchPort }} \
	--their-addr={{ .TheirHost }}:{{ .BenchPort }} \
	--num-conns={{ .Def.NumConns }} \
	--max-pending-calls={{ .Def.MaxPendingCalls }} \
	--num-iterations={{ .Def.NumIterations }} \
	--bytes-per-pay={{ .Def.BytesPerPay }} \
	--no-echo={{ .Def.NoEcho }} \
	{{ .Suffix }} 2> {{ .LogName }} | tee {{ .ReportName }}
`
	return textutil.Apply(ctx, format, textutil.M{
		"LogLevel":         "debug",
		"Prefix":           "perf stat -e cache-misses,cache-references,L1-dcache-loads,L1-dcache-load-misses,L1-icache-load-misses,LLC-loads,LLC-load-misses,branch-misses,page-faults",
		"Def":              def,
		"NodeIndex":        index,
		"Seed":             rand.Int63(),
		"ChuckdPath":       chuckdPath,
		"MetricsPort":      def.MetricsPort,
		"BenchPort":        def.BenchPort,
		"Suffix":           cmdSuffix,
		"TheirHost":        theirHost,
		"DoSend":           doSend,
		"CPUProfileName":   "bench-echo-cpu-" + ourHost + ".prof",
		"MemProfileName":   "bench-echo-mem-" + ourHost + ".prof",
		"BlockProfileName": "bench-echo-block-" + ourHost + ".prof",
		"TraceProfileName": "bench-echo-trace-" + ourHost + ".prof",
		"LogName":          "bench-echo-" + ourHost + ".log",
		"ReportName":       "bench-echo-" + ourHost + ".report",
	})
}
