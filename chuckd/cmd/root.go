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
	_ "github.com/groundhogchain/chuckd/runtime"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
)

func init() {
	cobra.OnInitialize(initAfterFlags)

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file (default is $HOME/"+api.LocalAgentPath+"/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "run in developer mode with verbose logging")
	rootCmd.PersistentFlags().StringVarP(&cpuProfilePath, "cpu", "p", "", "cpu profile path")
	rootCmd.PersistentFlags().StringVarP(&memProfilePath, "mem", "m", "", "memory profile path")

	rootCmd.AddCommand(serverCmd)
}

func initAfterFlags() {
	LoadConfig(ctx)
	if verbose {
		log.LogVerbosely()
	} else {
		log.LogQuietly()
	}
	log.Infof("chuckd version: %v, os: %v, arch: %v", ghc.Short(), runtime.GOOS, runtime.GOARCH)
	api.SetCPUProfilePath(cpuProfilePath)
	api.SetMemProfilePath(memProfilePath)
}

func LoadConfig(ctx context.Context) {

	// search for config in ~/.chuckd/chuckd.yaml
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("looking for home dir (%v)", err)
		}
		must.NoError(ctx, err)
		configPath = filepath.Join(home, api.LocalAgentPath, api.LocalAgentConfigFilebase)
	}
	log.Infof("loading config %v", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("reading config file (%v)", err)
	}

	var config api.Config
	err = yaml.Unmarshal(data, &config)
	must.NoError(ctx, err)

	cfg = &config
}

var (
	cfg *api.Config
)

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
