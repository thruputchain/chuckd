package cmd

import (
	benchbroadcast "github.com/groundhogchain/chuckd/bench/broadcast"
	"github.com/groundhogchain/chuckd/bench/testcluster"
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libmetrics"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
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
			return api.Invoke(ctx, func() {
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench testcluster"),
					attribute.Int64("tps", benchTPS),
				)
				cfg := LoadConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				srv := testcluster.NewTestClusterSystem(
					ctx,
					libtime.RealTime{},
					runtime.NewRandFromRoot(),
					*cfg,
					libnet.TCPNet{},
					benchTPS,
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
			return api.Invoke(ctx, func() {
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench broadcast"),
					attribute.Int("tree_fan_out", benchTreeFanOut),
					attribute.Int("payload_size", benchPaySize),
					attribute.Int("payloads_per_channel", benchPayPerChannel),
					attribute.Int("num_iterations", benchNumIterations),
				)
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
				select {} // hang
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
	benchTPS           int64
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
	// benchTestClusterCmd.Flags().StringVar(&benchInAddress, "in_addr", "localhost:7888", "bind address for internal apis")
	benchTestClusterCmd.Flags().StringVar(&benchOutAddress, "out_addr", "localhost:9111", "bind address for external apis")
	benchTestClusterCmd.Flags().StringVar(&benchWalletAddress, "w_addr", "localhost:6666", "bind address for wallet api")
	benchTestClusterCmd.Flags().Int64Var(&benchTPS, "tps", 10000, "transactions per second")
}
