package cmd

import (
	"time"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/scp/solo/leader"
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
				srv := leader.NewSoloServerService(
					ctx,
					libnet.TCPNet{},
					leader.UnionCombiner{},
					time.Millisecond*time.Duration(soloSlotTimeoutMillisecond),
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
	soloServerAddress          string
	soloSlotTimeoutMillisecond int64
)

func init() {
	rootCmd.AddCommand(soloCmd)

	soloCmd.AddCommand(soloStartCmd)
	soloStartCmd.Flags().StringVar(&soloServerAddress, "addr", "localhost:9111", "address for solo server api")
	soloStartCmd.Flags().Int64Var(&soloSlotTimeoutMillisecond, "timeout", 100, "slot timeout in milliseconds")
}
