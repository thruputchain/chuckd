package cmd

import (
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/server"
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
				srv := server.NewServer(
					ctx,
					runtime.NewRandFromRoot(),
					*cfg.Validator_config,
					serverTreeFanOut,
				)
				srv.Start(ctx, serverInternalServerAddress, serverExternalServerAddress)
				<-(chan int)(nil) // hang
			})
		},
	}
)

var (
	serverInternalServerAddress string
	serverExternalServerAddress string
	serverTreeFanOut            int // maximum fanout for multicast trees
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.AddCommand(serverStartCmd)
	serverStartCmd.Flags().StringVar(&serverInternalServerAddress, "inaddr", "localhost:7888", "bind address for internal apis")
	serverStartCmd.Flags().StringVar(&serverExternalServerAddress, "exaddr", "localhost:9111", "bind address for external apis")
	serverStartCmd.Flags().IntVar(&serverTreeFanOut, "fanout", 5, "maximum fanout of multicast trees")
}
