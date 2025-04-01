package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/spf13/cobra"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "tools to manipulate validator configuration",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	configInitCmd = &cobra.Command{
		Use:   "init",
		Short: "initialize identity and create barebones config",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := protoutil.InitConfig(ctx)
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
		},
	}
)

var (
	configToStdout bool
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configInitCmd)
	configInitCmd.Flags().BoolVar(&configToStdout, "stdout", false, "output to stdout")
}
