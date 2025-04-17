package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	ghc "github.com/groundhogchain/chuckd"
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/log"
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
	configPath     string
	verbose        bool
	cpuProfilePath string
	memProfilePath string
	seed           int64
)

func init() {
	cobra.OnInitialize(initAfterFlags)

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "",
		"config file (default is $HOME/"+api.LocalAgentPath+"/"+api.LocalAgentConfigFilebase+")")
	rootCmd.PersistentFlags().Int64VarP(&seed, "seed", "s", 0x0102030405060708, "randomness seed")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "run in developer mode with verbose logging")
	rootCmd.PersistentFlags().StringVarP(&cpuProfilePath, "cpu", "p", "", "cpu profile path")
	rootCmd.PersistentFlags().StringVarP(&memProfilePath, "mem", "m", "", "memory profile path")
}

func initAfterFlags() {
	if verbose {
		log.LogVerbosely()
	} else {
		log.LogQuietly()
	}
	log.Infof("chuckd version: %v, os: %v, arch: %v", ghc.Short(), runtime.GOOS, runtime.GOARCH)
	api.SetCPUProfilePath(cpuProfilePath)
	api.SetMemProfilePath(memProfilePath)
	chuckdruntime.SetRootSeed(seed)
}

func effectiveConfigPath(ctx context.Context) string {
	// search for config in ~/.chuckd/chuckd.json
	cfgPath := configPath
	if cfgPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("looking for home dir (%v)", err)
		}
		must.NoError(ctx, err)
		cfgPath = filepath.Join(home, api.LocalAgentPath, api.LocalAgentConfigFilebase)
	}
	return cfgPath
}

func LoadConfig(ctx context.Context) *proto.ValidatorConfig {

	cfgPath := effectiveConfigPath(ctx)
	log.Infof("loading config %v", cfgPath)

	data, err := os.ReadFile(cfgPath)
	must.Assertf(ctx, err == nil, "reading config file (%w)", err)

	var config proto.ValidatorConfig
	err = xdrutil.JsonToXdr(&config, data)
	must.NoError(ctx, err)

	return &config
}

func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return 0
}

func ExecuteWithConfig(cfgPath string) int {
	configPath = cfgPath
	return Execute()
}
