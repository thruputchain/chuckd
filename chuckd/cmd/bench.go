package cmd

import (
	benchbroadcast "github.com/groundhogchain/chuckd/bench/broadcast"
	"github.com/groundhogchain/chuckd/bench/loadgen"
	"github.com/groundhogchain/chuckd/bench/testcluster"
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libmetrics"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/quic"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/test/testnet"
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

	benchLoadGenCmd = &cobra.Command{
		Use:   "loadgen",
		Short: "start load generator benchmark server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench loadgen"),
					attribute.Int("txs_per_block", benchTPB),
				)
				srv := loadgen.NewLoadGenBench(
					ctx,
					runtime.NewRandFromRoot(),
					libnet.TCPNet{},
					benchTPB,
					benchCallsPerExec,
					benchBatchSize,
				)
				srv.Start(ctx, benchInAddress)
				select {}
			})
		},
	}

	benchLoadGenTestCmd = &cobra.Command{
		Use:   "loadgen-test",
		Short: "test the load generator",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				testCfg := testcluster.TestClusterConfig{
					TPS:          0,
					NumDb:        benchNumDb,
					NumExec:      benchNumExec,
					DbBasePort:   benchDbBasePort,
					ExecBasePort: benchExecBasePort,
				}
				var net libnet.Network
				if benchTCP {
					net = libnet.TCPNet{}
				} else if benchQUIC {
					net = quic.NewQUICNet(ctx)
				} else {
					net = testnet.NewTestNetwork()
				}
				loadgen.TestLoadGen(
					ctx,
					net,
					testCfg,
					benchInPort,
					benchTPB,
					benchCallsPerExec,
					benchBatchSize,
				)
			})
		},
	}

	benchTestClusterCmd = &cobra.Command{
		Use:   "testcluster",
		Short: "start testcluster benchmark server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench testcluster"),
					attribute.Int64("txs_per_sec", benchTPS),
				)
				cfg := LoadConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				outside := libnet.Network(libnet.TCPNet{})
				if benchQUIC {
					outside = quic.NewQUICNet(ctx)
				}
				testCfg := testcluster.TestClusterConfig{
					TPS:          benchTPS,
					NumDb:        benchNumDb,
					NumExec:      benchNumExec,
					DbBasePort:   benchDbBasePort,
					ExecBasePort: benchExecBasePort,
				}
				srv := testcluster.NewTestClusterSystem(
					ctx,
					libtime.RealTime{},
					runtime.NewRandFromRoot(),
					outside,
					*cfg,
					testCfg,
					benchCallsPerExec,
					benchBatchSize,
				)
				srv.Start(ctx, benchInAddress, benchOutAddress, benchWalletAddress)
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
				outside := libnet.Network(libnet.TCPNet{})
				if benchQUIC {
					outside = quic.NewQUICNet(ctx)
				}
				srv := benchbroadcast.NewBroadcastBenchmark(
					ctx,
					runtime.NewRandFromRoot(),
					outside,
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
	benchQUIC          bool
	benchServerAddress string
	benchTreeFanOut    int
	benchPaySize       int
	benchPayPerChannel int
	benchNumIterations int
	//
	benchInAddress     string
	benchOutAddress    string
	benchWalletAddress string
	benchTPS           int64
	benchTPB           int
	//
	benchNumDb        int
	benchNumExec      int
	benchCallsPerExec int
	benchBatchSize    int
	//
	benchInPort       int
	benchDbBasePort   int
	benchExecBasePort int
	benchTCP          bool // use TCP instead of testnet
)

func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.AddCommand(benchBroadcastCmd)
	benchBroadcastCmd.Flags().BoolVar(&benchQUIC, "quic", false, "use QUIC for external apis")
	benchBroadcastCmd.Flags().StringVar(&benchServerAddress, "addr", "localhost:9111", "address for broadcast api")
	benchBroadcastCmd.Flags().IntVar(&benchTreeFanOut, "fanout", 5, "fanout of multicast trees")
	benchBroadcastCmd.Flags().IntVar(&benchPaySize, "paysize", 1000*1000*10, "payload size")
	benchBroadcastCmd.Flags().IntVar(&benchPayPerChannel, "payperchan", 100, "payloads per channel")
	benchBroadcastCmd.Flags().IntVar(&benchNumIterations, "iterations", 1, "number of iterations")

	benchCmd.AddCommand(benchTestClusterCmd)
	benchTestClusterCmd.Flags().BoolVar(&benchQUIC, "quic", false, "use QUIC for external apis")
	benchTestClusterCmd.Flags().StringVar(&benchInAddress, "in_addr", "localhost:7888", "bind address for internal apis")
	benchTestClusterCmd.Flags().StringVar(&benchOutAddress, "out_addr", "localhost:9111", "bind address for external apis")
	benchTestClusterCmd.Flags().StringVar(&benchWalletAddress, "w_addr", "localhost:6666", "bind address for wallet api")
	benchTestClusterCmd.Flags().Int64Var(&benchTPS, "tps", 1e4, "transactions per second")
	benchTestClusterCmd.Flags().IntVar(&benchCallsPerExec, "exec-par", 2, "concurrent calls per executor")
	benchTestClusterCmd.Flags().IntVar(&benchBatchSize, "exec-batch", 100, "batch size for executor")
	benchTestClusterCmd.Flags().IntVar(&benchNumDb, "numdb", 2, "number of db shards")
	benchTestClusterCmd.Flags().IntVar(&benchNumExec, "numexec", 20, "number of executors")
	benchTestClusterCmd.Flags().IntVar(&benchDbBasePort, "dbport", 30000, "dbs base port")
	benchTestClusterCmd.Flags().IntVar(&benchExecBasePort, "execport", 40000, "execs base port")

	benchCmd.AddCommand(benchLoadGenCmd)
	benchLoadGenCmd.Flags().StringVar(&benchInAddress, "in_addr", "localhost:7888", "bind address for internal apis")
	benchLoadGenCmd.Flags().IntVar(&benchTPB, "tpb", 1e6, "transactions per block")
	benchLoadGenCmd.Flags().IntVar(&benchCallsPerExec, "exec-par", 2, "concurrent calls per executor")
	benchLoadGenCmd.Flags().IntVar(&benchBatchSize, "exec-batch", 100, "batch size for executor")

	benchCmd.AddCommand(benchLoadGenTestCmd)
	benchLoadGenTestCmd.Flags().IntVar(&benchTPB, "tpb", 1e6, "transactions per block")
	benchLoadGenTestCmd.Flags().IntVar(&benchNumDb, "numdb", 2, "number of db shards")
	benchLoadGenTestCmd.Flags().IntVar(&benchNumExec, "numexec", 20, "number of executors")
	benchLoadGenTestCmd.Flags().IntVar(&benchCallsPerExec, "exec-par", 2, "concurrent calls per executor")
	benchLoadGenTestCmd.Flags().IntVar(&benchBatchSize, "exec-batch", 100, "batch size for executor")
	benchLoadGenTestCmd.Flags().IntVar(&benchInPort, "inport", 54054, "coodinator port")
	benchLoadGenTestCmd.Flags().IntVar(&benchDbBasePort, "dbport", 30000, "dbs base port")
	benchLoadGenTestCmd.Flags().IntVar(&benchExecBasePort, "execport", 40000, "execs base port")
	benchLoadGenTestCmd.Flags().BoolVar(&benchTCP, "tcp", false, "use tcp")
	benchLoadGenTestCmd.Flags().BoolVar(&benchQUIC, "quic", false, "use quic")
}
