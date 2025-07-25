package cmd

import (
	"fmt"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/quic"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/system/dogwood"
	"github.com/spf13/cobra"
)

var (
	sysCmd = &cobra.Command{
		Use:   "sys",
		Short: "systems control",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	sysCoordinatorCmd = &cobra.Command{
		Use:   "coordinator",
		Short: "start coordinator system",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				cfg := LoadCoordinatorConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				inside := libnet.Network(libnet.TCPNet{})
				outside := libnet.Network(libnet.TCPNet{})
				if serverQUIC {
					inside = libnet.TCPNet{}
					outside = quic.NewQUICNet(ctx)
				}
				srv := coordinator.NewSystem(
					ctx,
					runtime.NewRandFromRoot(),
					cfg,
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

	sysDogwoodCmd = &cobra.Command{
		Use:   "dogwood",
		Short: "start dogwood (standalone disseminator) system",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				cfg := LoadDogwoodConfig(ctx)
				srv := dogwood.NewSystem(
					ctx,
					runtime.NewRandFromRoot(),
					libnet.Network(libnet.TCPNet{}),
					cfg,
				)
				srv.Start(ctx, serverOutAddress, serverInAddress)
				fmt.Println("dogwood is ready")
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
	rootCmd.AddCommand(sysCmd)

	sysCmd.AddCommand(sysCoordinatorCmd)
	sysCoordinatorCmd.Flags().BoolVar(&serverQUIC, "quic", false, "use QUIC for external apis")
	sysCoordinatorCmd.Flags().StringVar(&serverInAddress, "in-addr", "localhost:7888", "bind address for internal apis")
	sysCoordinatorCmd.Flags().StringVar(&serverOutAddress, "out-addr", "localhost:9111", "bind address for external apis")
	sysCoordinatorCmd.Flags().StringVar(&serverWalletAddress, "w-addr", "localhost:6666", "bind address for wallet api")
	sysCoordinatorCmd.Flags().IntVar(&serverCallsPerExec, "exec-par", 2, "concurrent calls per executor")
	sysCoordinatorCmd.Flags().IntVar(&serverBatchSize, "exec-batch", 100, "batch size for executor")

	sysCmd.AddCommand(sysDogwoodCmd)
	sysDogwoodCmd.Flags().StringVar(&serverInAddress, "in-addr", "localhost:10101", "address for api")
	sysDogwoodCmd.Flags().StringVar(&serverOutAddress, "out-addr", ":33055", "address for peers")
}
