package cmd

import (
	"github.com/groundhogchain/chuckd/chuckd/server"
	"github.com/spf13/cobra"
)

var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "coordinator server control",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	serverStartCmd = &cobra.Command{
		Use:   "start",
		Short: "start coordinator server",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := LoadConfig(ctx)
			srv := server.NewServer(
				ctx,
				serverInternalBindAddress,
				serverExternalBindAddress,
				cfg.Private_info,
				cfg.Quorum_slices,
			)
			srv.Start(ctx)
			<-(chan int)(nil) // hang
		},
	}
)

var (
	serverInternalBindAddress string
	serverExternalBindAddress string
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.AddCommand(serverStartCmd)
	serverStartCmd.Flags().StringVar(&serverInternalBindAddress, "inaddr", "localhost:7888", "bind address for internal apis")
	serverStartCmd.Flags().StringVar(&serverExternalBindAddress, "exaddr", "localhost:9111", "bind address for external apis")
}
