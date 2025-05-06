package cmd

import (
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system"
	"github.com/spf13/cobra"
)

var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "coordinator server control",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	serverStartCmd = &cobra.Command{
		Use:   "start",
		Short: "start coordinator server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				cfg := LoadConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				srv := system.NewSystem(
					ctx,
					runtime.NewRandFromRoot(),
					*cfg,
				)
				srv.Start(ctx, serverInternalAddress, serverExternalAddress, serverWalletAddress)
				<-(chan int)(nil) // hang
			})
		},
	}
)

var (
	serverInternalAddress string
	serverExternalAddress string
	serverWalletAddress   string
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.AddCommand(serverStartCmd)
	serverStartCmd.Flags().StringVar(&serverInternalAddress, "inaddr", "localhost:7888", "bind address for internal apis")
	serverStartCmd.Flags().StringVar(&serverExternalAddress, "exaddr", "localhost:9111", "bind address for external apis")
	serverStartCmd.Flags().StringVar(&serverWalletAddress, "waddr", "localhost:6666", "bind address for wallet api")
}
