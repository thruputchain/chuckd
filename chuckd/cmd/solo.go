package cmd

import (
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/scp/solo/server"
	"github.com/spf13/cobra"
)

var (
	soloCmd = &cobra.Command{
		Use:   "solo",
		Short: "manage the solo scp server",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	soloStartCmd = &cobra.Command{
		Use:   "start",
		Short: "start the solo scp server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				cfg := LoadConfig(ctx)
				srv := server.NewSoloServerService(
					ctx,
					libnet.TCPNet{},
					cfg.Private_info,
					cfg.Known_validators,
				)
				srv.Start(ctx, soloServerAddress)
				<-(chan int)(nil) // hang
			})
		},
	}
)

var (
	soloServerAddress string
)

func init() {
	rootCmd.AddCommand(soloCmd)

	soloCmd.AddCommand(soloStartCmd)
	soloStartCmd.Flags().StringVar(&soloServerAddress, "addr", "localhost:9111", "address for solo server api")
}
