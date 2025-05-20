package cmd

import (
	benchbroadcast "github.com/groundhogchain/chuckd/bench/broadcast"
	"github.com/groundhogchain/chuckd/bench/testcluster"
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/spf13/cobra"
)

var (
	benchCmd = &cobra.Command{
		Use:   "bench",
		Short: "run server in benchmark mode",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	benchTestClusterCmd = &cobra.Command{
		Use:   "testcluster",
		Short: "start testcluster benchmark server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				cfg := LoadConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				srv := testcluster.NewTestClusterSystem(
					ctx,
					runtime.NewRandFromRoot(),
					*cfg,
					libnet.TCPNet{},
				)
				srv.Start(ctx, benchOutAddress, benchWalletAddress)
				select {}
			})
		},
	}

	benchBroadcastCmd = &cobra.Command{
		Use:   "broadcast",
		Short: "start broadcast benchmark server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				cfg := LoadConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				srv := benchbroadcast.NewBroadcastBenchmark(
					ctx,
					runtime.NewRandFromRoot(),
					*cfg.Validator_config,
					benchTreeFanOut,
					benchPaySize,
					benchPayPerChannel,
					benchNumIterations,
				)
				srv.Start(ctx, benchServerAddress)
				<-(chan int)(nil) // hang
			})
		},
	}
)

var (
	benchServerAddress string
	benchTreeFanOut    int
	benchPaySize       int
	benchPayPerChannel int
	benchNumIterations int
	//
	// benchInAddress     string
	benchOutAddress    string
	benchWalletAddress string
)

func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.AddCommand(benchBroadcastCmd)
	benchBroadcastCmd.Flags().StringVar(&benchServerAddress, "addr", "localhost:9111", "address for broadcast api")
	benchBroadcastCmd.Flags().IntVar(&benchTreeFanOut, "fanout", 5, "fanout of multicast trees")
	benchBroadcastCmd.Flags().IntVar(&benchPaySize, "paysize", 1000*1000*10, "payload size")
	benchBroadcastCmd.Flags().IntVar(&benchPayPerChannel, "payperchan", 100, "payloads per channel")
	benchBroadcastCmd.Flags().IntVar(&benchNumIterations, "iterations", 1, "number of iterations")

	benchCmd.AddCommand(benchTestClusterCmd)
	// benchTestClusterCmd.Flags().StringVar(&benchInAddress, "inaddr", "localhost:7888", "bind address for internal apis")
	benchTestClusterCmd.Flags().StringVar(&benchOutAddress, "exaddr", "localhost:9111", "bind address for external apis")
	benchTestClusterCmd.Flags().StringVar(&benchWalletAddress, "waddr", "localhost:6666", "bind address for wallet api")
}
