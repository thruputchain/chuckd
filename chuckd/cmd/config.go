package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/spf13/cobra"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "tools to manipulate validator configuration",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	configInitCmd = &cobra.Command{
		Use:   "init",
		Short: "initialize identity and create barebones config",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				cfg := protoutil.InitConfig(ctx, runtime.NewCryptoRand(ctx), "")
				if configToStdout {
					fmt.Println(form.SprintJSON(cfg))
				} else {
					cfgPath := effectiveConfigPath(ctx)
					_, err := os.Stat(cfgPath)
					if errors.Is(err, os.ErrNotExist) {
						err := os.WriteFile(cfgPath, form.SprintJSONBytes(cfg), 0644)
						must.NoError(ctx, err)
					} else {
						must.NoError(ctx, err)
						must.Errorf(ctx, "config already exists: %s", cfgPath)
					}
				}
			})
		},
	}

	configGroupCmd = &cobra.Command{
		Use:   "group",
		Short: "generate a group of configs",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				cfgs := protoutil.GenerateConfigs(ctx, runtime.NewCryptoRand(ctx), configGroupSize)
				protoutil.WriteConfigsToDir(ctx, cfgs, configOutputDir)
			})
		},
	}
)

var (
	configToStdout  bool
	configGroupSize int
	configOutputDir string
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configInitCmd)
	configInitCmd.Flags().BoolVar(&configToStdout, "stdout", false, "output to stdout")

	configCmd.AddCommand(configGroupCmd)
	configGroupCmd.Flags().IntVar(&configGroupSize, "size", 3, "size of group")
	configGroupCmd.Flags().StringVar(&configOutputDir, "outdir", ".", "output directory")
}
