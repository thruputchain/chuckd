package cmd

import (
	cryptorand "crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thruputchain/chuckd/chuckd/cmdkit"
	"github.com/thruputchain/chuckd/util/configtext"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
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
			return cmdkit.Invoke(ctx, func() {
				vcfg := protoutil.InitConfig(ctx, cryptorand.Reader, "")
				cfg := proto.Config{
					Validator_config: vcfg,
				}
				// The same operator representation LoadCoordinatorConfig reads:
				// base58 public keys, hex secret keys.
				text, err := configtext.Marshal(&cfg)
				must.NoError(ctx, err)
				if configToStdout {
					fmt.Println(string(text))
				} else {
					cfgPath := effectiveConfigPath(ctx)
					must.NoError(ctx, writeNewConfig(cfgPath, text))
				}
			})
		},
	}

	configGroupCmd = &cobra.Command{
		Use:   "group",
		Short: "generate a group of configs",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdkit.Invoke(ctx, func() {
				vcfgs := protoutil.GenerateConfigs(ctx, cryptorand.Reader, configGroupSize)
				cfgs := make([]*proto.Config, len(vcfgs))
				for i := range vcfgs {
					cfgs[i] = &proto.Config{
						Validator_config: vcfgs[i],
					}
				}
				protoutil.WriteConfigsToDir(ctx, cfgs, configOutputDir)
			})
		},
	}
)

// writeNewConfig creates path with its parent directory, readable by this
// user only (the file holds the validator's secret key), and refuses to
// replace an existing file.
func writeNewConfig(path string, text []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("config already exists: %s", path)
		}
		return err
	}
	if _, err := f.Write(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

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
