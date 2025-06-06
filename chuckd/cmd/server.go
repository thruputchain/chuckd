package cmd

import (
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/quic"
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
			return api.Invoke(ctx, func() {
				cfg := LoadConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				inside := libnet.Network(libnet.TCPNet{})
				outside := libnet.Network(libnet.TCPNet{})
				if serverQUIC {
					inside = libnet.TCPNet{}
					outside = quic.NewQUICNet(ctx)
				}
				srv := system.NewSystem(
					ctx,
					runtime.NewRandFromRoot(),
					*cfg,
					inside,
					outside,
					serverCallsPerExec,
					serverBatchSize,
				)
				srv.Start(ctx, serverInAddress, serverOutAddress, serverWalletAddress)
				select {} // hang
			})
		},
	}
)

var (
	serverQUIC          bool
	serverInAddress     string
	serverOutAddress    string
	serverWalletAddress string
	serverCallsPerExec  int
	serverBatchSize     int
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.AddCommand(serverStartCmd)
	serverStartCmd.Flags().BoolVar(&serverQUIC, "quic", false, "use QUIC for external apis")
	serverStartCmd.Flags().StringVar(&serverInAddress, "in-addr", "localhost:7888", "bind address for internal apis")
	serverStartCmd.Flags().StringVar(&serverOutAddress, "out-addr", "localhost:9111", "bind address for external apis")
	serverStartCmd.Flags().StringVar(&serverWalletAddress, "w-addr", "localhost:6666", "bind address for wallet api")
	serverStartCmd.Flags().IntVar(&serverCallsPerExec, "exec-par", 2, "concurrent calls per executor")
	serverStartCmd.Flags().IntVar(&serverBatchSize, "exec-batch", 100, "batch size for executor")
}
