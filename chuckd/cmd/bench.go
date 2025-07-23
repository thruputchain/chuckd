package cmd

import (
	benchbroadcast "github.com/groundhogchain/chuckd/bench/broadcast"
	benchdogwood "github.com/groundhogchain/chuckd/bench/dogwood"
	benchecho "github.com/groundhogchain/chuckd/bench/echo"
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
		Short: "run a benchmark",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	benchLoadGenCmd = &cobra.Command{
		Use:   "loadgen",
		Short: "start load generator benchmark",
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
					benchGenPar,
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
					benchGenPar,
					benchCallsPerExec,
					benchBatchSize,
				)
			})
		},
	}

	benchTestClusterCmd = &cobra.Command{
		Use:   "testcluster",
		Short: "start testcluster benchmark",
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
		Short: "start broadcast benchmark",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				cfg := LoadConfig(ctx)
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench broadcast"),
					attribute.Int("tree_fan_out", benchTreeFanOut),
					attribute.Int("chunk_size", int(cfg.Broadcast_config.Chunk_size)),
					attribute.Int("chunk_per_channel", benchChunkPerChannel),
					attribute.Int("num_iterations", benchNumIterations),
				)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				must.Assertf(ctx, cfg.Broadcast_config != nil, "missing broadcast config section")
				outside := libnet.Network(libnet.TCPNet{})
				if benchQUIC {
					outside = quic.NewQUICNet(ctx)
				}
				srv := benchbroadcast.NewBroadcastBenchmark(
					ctx,
					runtime.NewRandFromRoot(),
					outside,
					*cfg,
					benchTreeFanOut,
					benchChunkPerChannel,
					benchBytesPerSec,
					benchNumIterations,
				)
				srv.Start(ctx, benchServerAddress)
				select {} // hang
			})
		},
	}

	benchDogwoodCmd = &cobra.Command{
		Use:   "dogwood",
		Short: "start dogwood benchmark",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				cfg := LoadConfig(ctx)
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench dogwood"),
					attribute.Int("tree_fan_out", benchTreeFanOut),
					attribute.Int("bytes_per_pay", benchBytesPerPay),
					attribute.Int("num_iterations", benchNumIterations),
				)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				must.Assertf(ctx, cfg.Dogwood_config != nil, "missing dogwood config section")
				outside := libnet.Network(libnet.TCPNet{})
				if benchQUIC {
					outside = quic.NewQUICNet(ctx)
				}
				srv := benchdogwood.NewDogwoodBenchmark(
					ctx,
					runtime.NewRandFromRoot(),
					outside,
					*cfg,
					benchTreeFanOut,
					benchBytesPerPay,
					benchNumIterations,
				)
				srv.Start(ctx, benchServerAddress)
				srv.Wait()
			})
		},
	}

	benchEchoCmd = &cobra.Command{
		Use:   "echo",
		Short: "start echo benchmark",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {

				srv := benchecho.NewEchoBench(
					ctx,
					libnet.TCPNet{},
					benchNumConns,
					benchMaxPendingCalls,
					benchNumIterations,
					benchBytesPerPay,
					benchNoEcho,
				)
				srv.Run(ctx, benchOurAddress, benchTheirAddress)
			})
		},
	}
)

var (
	benchQUIC            bool
	benchServerAddress   string
	benchTreeFanOut      int
	benchChunkPerChannel int
	benchBytesPerSec     int
	benchNumIterations   int
	benchBytesPerPay     int
	//
	benchInAddress     string
	benchOutAddress    string
	benchWalletAddress string
	benchTPS           int64
	benchTPB           int
	//
	benchNumDb        int
	benchNumExec      int
	benchGenPar       int
	benchCallsPerExec int
	benchBatchSize    int
	//
	benchInPort       int
	benchDbBasePort   int
	benchExecBasePort int
	benchTCP          bool // use TCP instead of testnet
	//
	benchOurAddress      string
	benchTheirAddress    string
	benchNumConns        int
	benchMaxPendingCalls int
	benchNoEcho          bool
)

func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.AddCommand(benchBroadcastCmd)
	benchBroadcastCmd.Flags().BoolVar(&benchQUIC, "quic", false, "use QUIC for external apis")
	benchBroadcastCmd.Flags().StringVar(&benchServerAddress, "addr", "localhost:9111", "address for broadcast api")
	benchBroadcastCmd.Flags().IntVar(&benchTreeFanOut, "fan-out", 5, "fan-out of multicast trees")
	benchBroadcastCmd.Flags().IntVar(&benchChunkPerChannel, "chunk-per-chan", 100, "chunks per channel")
	benchBroadcastCmd.Flags().IntVar(&benchBytesPerSec, "bytes-per-sec", 10000, "bytes per second")
	benchBroadcastCmd.Flags().IntVar(&benchNumIterations, "num-iterations", 1, "number of iterations")

	benchCmd.AddCommand(benchDogwoodCmd)
	benchDogwoodCmd.Flags().BoolVar(&benchQUIC, "quic", false, "use QUIC for external apis")
	benchDogwoodCmd.Flags().StringVar(&benchServerAddress, "addr", "localhost:9111", "address for broadcast api")
	benchDogwoodCmd.Flags().IntVar(&benchTreeFanOut, "fan-out", 5, "fan-out of multicast trees")
	benchDogwoodCmd.Flags().IntVar(&benchBytesPerPay, "bytes-per-pay", 300000, "bytes per payload")
	benchDogwoodCmd.Flags().IntVar(&benchNumIterations, "num-iterations", 10000, "number of iterations")

	benchCmd.AddCommand(benchTestClusterCmd)
	benchTestClusterCmd.Flags().BoolVar(&benchQUIC, "quic", false, "use QUIC for external apis")
	benchTestClusterCmd.Flags().StringVar(&benchInAddress, "in-addr", "localhost:7888", "bind address for internal apis")
	benchTestClusterCmd.Flags().StringVar(&benchOutAddress, "out-addr", "localhost:9111", "bind address for external apis")
	benchTestClusterCmd.Flags().StringVar(&benchWalletAddress, "w-addr", "localhost:6666", "bind address for wallet api")
	benchTestClusterCmd.Flags().Int64Var(&benchTPS, "tps", 1e4, "transactions per second")
	benchTestClusterCmd.Flags().IntVar(&benchCallsPerExec, "exec-par", 2, "concurrent calls per executor")
	benchTestClusterCmd.Flags().IntVar(&benchBatchSize, "exec-batch", 100, "batch size for executor")
	benchTestClusterCmd.Flags().IntVar(&benchNumDb, "num-db", 2, "number of db shards")
	benchTestClusterCmd.Flags().IntVar(&benchNumExec, "num-exec", 20, "number of executors")
	benchTestClusterCmd.Flags().IntVar(&benchDbBasePort, "db-port", 30000, "dbs base port")
	benchTestClusterCmd.Flags().IntVar(&benchExecBasePort, "exec-port", 40000, "execs base port")

	benchCmd.AddCommand(benchLoadGenCmd)
	benchLoadGenCmd.Flags().StringVar(&benchInAddress, "in-addr", "localhost:7888", "bind address for internal apis")
	benchLoadGenCmd.Flags().IntVar(&benchTPB, "tpb", 1e6, "transactions per block")
	benchLoadGenCmd.Flags().IntVar(&benchGenPar, "gen-par", 20, "generation parallelism")
	benchLoadGenCmd.Flags().IntVar(&benchCallsPerExec, "exec-par", 2, "concurrent calls per executor")
	benchLoadGenCmd.Flags().IntVar(&benchBatchSize, "exec-batch", 100, "batch size for executor")

	benchCmd.AddCommand(benchLoadGenTestCmd)
	benchLoadGenTestCmd.Flags().IntVar(&benchTPB, "tpb", 1e6, "transactions per block")
	benchLoadGenTestCmd.Flags().IntVar(&benchNumDb, "num-db", 2, "number of db shards")
	benchLoadGenTestCmd.Flags().IntVar(&benchNumExec, "num-exec", 20, "number of executors")
	benchLoadGenTestCmd.Flags().IntVar(&benchGenPar, "gen-par", 20, "generation parallelism")
	benchLoadGenTestCmd.Flags().IntVar(&benchCallsPerExec, "exec-par", 2, "concurrent calls per executor")
	benchLoadGenTestCmd.Flags().IntVar(&benchBatchSize, "exec-batch", 100, "batch size for executor")
	benchLoadGenTestCmd.Flags().IntVar(&benchInPort, "in-port", 54054, "coodinator port")
	benchLoadGenTestCmd.Flags().IntVar(&benchDbBasePort, "db-port", 30000, "dbs base port")
	benchLoadGenTestCmd.Flags().IntVar(&benchExecBasePort, "exec-port", 40000, "execs base port")
	benchLoadGenTestCmd.Flags().BoolVar(&benchTCP, "tcp", false, "use tcp")
	benchLoadGenTestCmd.Flags().BoolVar(&benchQUIC, "quic", false, "use quic")

	benchCmd.AddCommand(benchEchoCmd)
	benchEchoCmd.Flags().StringVar(&benchOurAddress, "our-addr", ":8181", "our address")
	benchEchoCmd.Flags().StringVar(&benchTheirAddress, "their-addr", ":8181", "their address")
	benchEchoCmd.Flags().IntVar(&benchNumConns, "num-conns", 10, "number of connections")
	benchEchoCmd.Flags().IntVar(&benchMaxPendingCalls, "max-pending-calls", 200, "max pending calls")
	benchEchoCmd.Flags().IntVar(&benchNumIterations, "num-iterations", 500e3, "number of iterations")
	benchEchoCmd.Flags().IntVar(&benchBytesPerPay, "bytes-per-pay", 60e3, "bytes per payload")
	benchEchoCmd.Flags().BoolVar(&benchNoEcho, "no-echo", true, "don't echo the request payload")
}
