package cmd

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/thruputchain/chuckd/util/configtext"
	"github.com/rs/zerolog"
	ghc "github.com/thruputchain/chuckd"
	"github.com/thruputchain/chuckd/chuckd/cmdkit"
	chuckdruntime "github.com/thruputchain/chuckd/runtime"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/librpc"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
	"github.com/xdrpp/goxdr/rpc"

	"github.com/spf13/cobra"
)

var ctx = context.Background()

var (
	rootCmd = &cobra.Command{
		Use:     "chuckd",
		Short:   "chuckd is a command-line client for the groundhog coordinator server",
		Long:    ``,
		Version: ghc.Short(),
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
)

var (
	rootConfigPath         string
	rootCPUProfilePath     string
	rootTraceProfilePath   string
	rootMemProfilePath     string
	rootMutexProfilePath   string
	rootBlockProfilePath   string
	rootSeed               int64
	rootInstanceID         string
	rootMetricsAddr        string
	rootPprof              bool
	rootPprofAllowRemote   bool
	rootPprofMutexFraction int
	rootPprofBlockRate     int
	rootStackTrace         bool
	rootColor              bool
	rootLogLevel           string
	rootMaxMsgSize         int
	rootRpcLogCalls        bool
	rootRpcLogStats        bool
)

func init() {
	cobra.OnInitialize(initAfterFlags)
	// cobra.OnFinalize()

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().StringVarP(&rootConfigPath, "config", "c", "",
		"config file (default is $HOME/"+cmdkit.LocalAgentPath+"/"+cmdkit.LocalAgentConfigFilebase+")")
	rootCmd.PersistentFlags().Int64Var(&rootSeed, "seed", 0x0102030405060708, "randomness seed")
	rootCmd.PersistentFlags().StringVarP(&rootLogLevel, "level", "l", "info", "log level")
	rootCmd.PersistentFlags().BoolVar(&rootColor, "color", true, "colorize log output")
	rootCmd.PersistentFlags().StringVarP(&rootCPUProfilePath, "cpu", "p", "", "cpu profile path")
	rootCmd.PersistentFlags().StringVarP(&rootTraceProfilePath, "trace", "t", "", "trace profile path")
	rootCmd.PersistentFlags().StringVarP(&rootMemProfilePath, "mem", "m", "", "memory profile path")
	rootCmd.PersistentFlags().StringVarP(&rootMutexProfilePath, "mutex", "x", "", "mutex profile path")
	rootCmd.PersistentFlags().StringVarP(&rootBlockProfilePath, "block", "k", "", "block profile path")
	rootCmd.PersistentFlags().StringVarP(&rootInstanceID, "instance", "i", "", "instance id")
	rootCmd.PersistentFlags().BoolVarP(&rootStackTrace, "stack", "s", false, "stack trace on error")
	rootCmd.PersistentFlags().StringVar(&rootMetricsAddr, "metrics", ":2222", "metrics address")
	rootCmd.PersistentFlags().BoolVar(&rootPprof, "pprof", false, "enable pprof endpoints on the metrics listener (loopback only unless --pprof-allow-remote)")
	rootCmd.PersistentFlags().BoolVar(&rootPprofAllowRemote, "pprof-allow-remote", false, "with --pprof, serve unauthenticated pprof endpoints when the metrics listener is not loopback (or set GROUNDHOG_PPROF_ALLOW_REMOTE=1); pprof requests must use an IP-literal or localhost Host header (DNS-rebinding guard)")
	rootCmd.PersistentFlags().IntVar(&rootPprofMutexFraction, "pprof-mutex-fraction", 0, "mutex sampling fraction with --pprof (0 off; devnet: 5 samples one in five contention events)")
	rootCmd.PersistentFlags().IntVar(&rootPprofBlockRate, "pprof-block-rate", 0, "block sampling rate in nanoseconds with --pprof (0 off; start at 100000 and validate with coordinator_bench_3n before lowering)")
	rootCmd.PersistentFlags().IntVar(&rootMaxMsgSize, "max-msg-size", 100e6, "maximum message size")
	rootCmd.PersistentFlags().BoolVar(&rootRpcLogCalls, "rpc-log-calls", false, "log rpc call arguments and replies")
	rootCmd.PersistentFlags().BoolVar(&rootRpcLogStats, "rpc-log-stats", false, "log rpc call latencies")
}

func initAfterFlags() {
	level, err := zerolog.ParseLevel(rootLogLevel)
	if err != nil {
		log.Fatal(err)
	}
	liblog.SetLogLevel(level, rootColor)

	librpc.SetLogCalls(rootRpcLogCalls)
	librpc.SetLogStats(rootRpcLogStats)

	if rootStackTrace {
		cmdkit.SetStackTrace(true)
	}
	liblog.Infof(context.Background(), "chuckd version: %v, os: %v, arch: %v", ghc.Short(), runtime.GOOS, runtime.GOARCH)
	cmdkit.SetCPUProfilePath(rootCPUProfilePath)
	cmdkit.SetTraceProfilePath(rootTraceProfilePath)
	cmdkit.SetMemProfilePath(rootMemProfilePath)
	cmdkit.SetMutexProfilePath(rootMutexProfilePath)
	cmdkit.SetBlockProfilePath(rootBlockProfilePath)
	cmdkit.SetInstanceID(rootInstanceID)
	cmdkit.SetMetricsAddr(rootMetricsAddr)
	cmdkit.SetPprof(rootPprof, rootPprofMutexFraction, rootPprofBlockRate)
	cmdkit.SetPprofAllowRemote(rootPprofAllowRemote)
	rpc.DefaultMaxMsgSize = rootMaxMsgSize
	chuckdruntime.SetRootSeed(rootSeed)
}

func effectiveConfigPath(ctx context.Context) string {
	// search for config in ~/.chuckd/chuckd.json
	cfgPath := rootConfigPath
	if cfgPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			liblog.Fatalf(ctx, "looking for home dir (%v)", err)
		}
		must.NoError(ctx, err)
		cfgPath = filepath.Join(home, cmdkit.LocalAgentPath, cmdkit.LocalAgentConfigFilebase)
	}
	return cfgPath
}

func LoadCoordinatorConfig(ctx context.Context) proto.Config {

	cfgPath := effectiveConfigPath(ctx)
	liblog.Infof(ctx, "loading config %v", cfgPath)

	data, err := os.ReadFile(cfgPath)
	must.Assertf(ctx, err == nil, "reading config file (%w)", err)

	var config proto.Config
	err = configtext.Unmarshal(data, &config)
	must.NoError(ctx, err)

	liblog.Infof(ctx, "loaded coordinator config")

	return config
}

func LoadDogwoodConfig(ctx context.Context) proto.DogwoodStandaloneConfig {

	cfgPath := effectiveConfigPath(ctx)
	liblog.Infof(ctx, "loading config %v", cfgPath)

	data, err := os.ReadFile(cfgPath)
	must.Assertf(ctx, err == nil, "reading config file (%w)", err)

	var config proto.DogwoodStandaloneConfig
	err = configtext.Unmarshal(data, &config)
	must.NoError(ctx, err)

	return config
}

func Execute() error {
	return rootCmd.Execute()
}
