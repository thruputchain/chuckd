package cmd

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	ghc "github.com/groundhogchain/chuckd"
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	chuckdruntime "github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"

	"github.com/spf13/cobra"
)

var ctx = context.Background()

var (
	rootCmd = &cobra.Command{
		Use:   "chuckd",
		Short: "chuckd is a command-line client for the groundhog coordinator server",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
)

var (
	rootConfigPath     string
	rootVerbose        bool
	rootCPUProfilePath string
	rootMemProfilePath string
	rootSeed           int64
	rootInstanceID     string
	rootMetricsAddr    string
	rootStackTrace     bool
)

func init() {
	cobra.OnInitialize(initAfterFlags)
	// cobra.OnFinalize()

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().StringVarP(&rootConfigPath, "config", "c", "",
		"config file (default is $HOME/"+api.LocalAgentPath+"/"+api.LocalAgentConfigFilebase+")")
	rootCmd.PersistentFlags().Int64Var(&rootSeed, "seed", 0x0102030405060708, "randomness seed")
	rootCmd.PersistentFlags().BoolVarP(&rootVerbose, "verbose", "v", false, "run in developer mode with verbose logging")
	rootCmd.PersistentFlags().StringVarP(&rootCPUProfilePath, "cpu", "p", "", "cpu profile path")
	rootCmd.PersistentFlags().StringVarP(&rootMemProfilePath, "mem", "m", "", "memory profile path")
	rootCmd.PersistentFlags().StringVarP(&rootInstanceID, "instance", "i", "", "instance id")
	rootCmd.PersistentFlags().BoolVarP(&rootStackTrace, "stack", "s", false, "stack trace on error")
	rootCmd.PersistentFlags().StringVar(&rootMetricsAddr, "metrics", ":2222", "metrics address")
}

func initAfterFlags() {
	if rootVerbose {
		liblog.LogVerbosely()
	} else {
		liblog.LogQuietly()
	}
	if rootStackTrace {
		api.SetStackTrace(true)
	}
	liblog.Infof("chuckd version: %v, os: %v, arch: %v", ghc.Short(), runtime.GOOS, runtime.GOARCH)
	api.SetCPUProfilePath(rootCPUProfilePath)
	api.SetMemProfilePath(rootMemProfilePath)
	api.SetInstanceID(rootInstanceID)
	api.SetMetricsAddr(rootMetricsAddr)
	chuckdruntime.SetRootSeed(rootSeed)
}

func effectiveConfigPath(ctx context.Context) string {
	// search for config in ~/.chuckd/chuckd.json
	cfgPath := rootConfigPath
	if cfgPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			liblog.Fatalf("looking for home dir (%v)", err)
		}
		must.NoError(ctx, err)
		cfgPath = filepath.Join(home, api.LocalAgentPath, api.LocalAgentConfigFilebase)
	}
	return cfgPath
}

func LoadConfig(ctx context.Context) *proto.Config {

	cfgPath := effectiveConfigPath(ctx)
	liblog.Infof("loading config %v", cfgPath)

	data, err := os.ReadFile(cfgPath)
	must.Assertf(ctx, err == nil, "reading config file (%w)", err)

	var config proto.Config
	err = xdrutil.JsonToXdr(&config, data)
	must.NoError(ctx, err)

	return &config
}

func Execute() error {
	return rootCmd.Execute()
}
