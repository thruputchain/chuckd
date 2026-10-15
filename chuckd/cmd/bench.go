package cmd

import (
	"fmt"
	"os/exec"
	"path"

	"github.com/thruputchain/chuckd/util/encoding"
	"github.com/spf13/cobra"
	"github.com/thruputchain/chuckd/chuckd/cmdkit"
	"github.com/thruputchain/chuckd/runtime"
	"github.com/thruputchain/chuckd/sub/loadgen"
	"github.com/thruputchain/chuckd/system/coordinator"
	"github.com/thruputchain/chuckd/system/coordinator_bench"
	"github.com/thruputchain/chuckd/system/coordinator_loadgen"
	"github.com/thruputchain/chuckd/system/dogwood_bench"
	"github.com/thruputchain/chuckd/system/echo_bench"
	"github.com/thruputchain/chuckd/system/scheduler"
	"github.com/thruputchain/chuckd/system/systemkit"
	"github.com/thruputchain/chuckd/test/testnet"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libmetrics"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/libtime"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
	"go.opentelemetry.io/otel/attribute"
)

var (
	benchCmd = &cobra.Command{
		Use:   "bench",
		Short: "run a benchmark",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	benchCoordinatorCmd = &cobra.Command{
		Use:   "coordinator",
		Short: "start coordinator benchmark",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdkit.Invoke(ctx, func() {
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench coordinator"),
					attribute.Int64("txs_per_sec", benchTxsPerSec),
				)
				cfg := LoadCoordinatorConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				outside := libnet.TCPReuseNet{}
				testCfg := coordinator_bench.CoordinatorBenchConfig{
					TxsPerSec: benchTxsPerSec,
					SchedPort: benchSchedPort,
				}
				sys := coordinator_bench.NewCoordinatorBenchSystem(
					ctx,
					libtime.RealTime{},
					runtime.NewRandFromRoot(),
					outside,
					cfg,
					testCfg,
					path.Join(benchStorageDir, "db"),
				)
				opcfg := coordinator.OperationalConfig{
					StorageDir:     benchStorageDir,
					SchedAddr:      fmt.Sprintf("localhost:%d", benchSchedPort),
					DebugPort:      benchDebugPort,
					InAddr:         benchInAddress,
					OutAddr:        benchOutAddress,
					SCPDaemonPath:  benchSCPDaemonPath,
					SCPHandleAddr:  benchSCPHandleAddr,
					SCPControlAddr: benchSCPControlAddr,
					IsValidator:    benchSCPIsValidator,
				}
				ready := make(chan struct{})
				stopAfterNumCommits := &benchNumCommits
				if benchNumCommits <= 0 {
					stopAfterNumCommits = nil
				}
				sys.Start(ctx, opcfg, stopAfterNumCommits, nil, ready, nil)
				<-ready
				liblog.Infof(ctx, "coordinator benchmark exiting")
			})
		},
	}

	benchCoordinatorLoadgenCmd = &cobra.Command{
		Use:   "coordinator_loadgen",
		Short: "start coordinator with loadgen benchmark",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			networkHash, err := encoding.Decode32(benchNetworkHash)
			if err != nil {
				return fmt.Errorf("network hash: %w", err)
			}
			return cmdkit.Invoke(ctx, func() {
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench coordinator"),
					attribute.Int("txs_per_block", benchTxsPerBlock),
				)
				cfg := LoadCoordinatorConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				testCfg := loadgen.Config{
					NumAccts:       benchNumAccts,
					NumTxsPerBlock: benchTxsPerBlock,
					NumBlocks:      benchNumBlocks,
					NetworkHash:    networkHash,
				}
				// pick scheduler system
				var inside libnet.Network
				var schedSystem systemkit.SchedulerSystem
				if benchMockScheduler {
					inside = testnet.NewTestNetwork()
					schedSystem = scheduler.NewMockScheduler(ctx, inside, cfg.Validator_config.Private_info, path.Join(benchStorageDir, "db"))
				} else {
					inside = libnet.TCPReuseNet{}
					schedSystem = scheduler.NewLiveScheduler()
				}
				// coordinator
				sys := coordinator_loadgen.NewCoordinatorLoadgenSystem(
					ctx,
					libtime.RealTime{},
					runtime.NewRandFromRoot(),
					inside,
					libnet.TCPReuseNet{},
					cfg,
					testCfg,
					schedSystem,
					benchSkip,
				)
				opcfg := coordinator.OperationalConfig{
					StorageDir:     benchStorageDir,
					SchedAddr:      benchSchedAddr,
					DebugPort:      benchDebugPort,
					InAddr:         benchInAddress,
					OutAddr:        benchOutAddress,
					SCPDaemonPath:  benchSCPDaemonPath,
					SCPHandleAddr:  benchSCPHandleAddr,
					SCPControlAddr: benchSCPControlAddr,
					IsValidator:    benchSCPIsValidator,
				}
				ready := make(chan struct{})
				sys.Start(ctx, opcfg, benchLoadgenAddr, ready)
				<-ready
				sys.Close() // kills scpd
			})
		},
	}

	benchDogwoodCmd = &cobra.Command{
		Use:   "dogwood",
		Short: "start dogwood benchmark",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdkit.Invoke(ctx, func() {
				cfg := LoadCoordinatorConfig(ctx)
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench dogwood"),
					attribute.Int("bytes_per_pay", benchBytesPerPay),
					attribute.Int("num_iterations", benchNumIterations),
				)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				must.Assertf(ctx, cfg.Dogwood_config != nil, "missing dogwood config section")
				outside := libnet.Network(libnet.TCPNet{})
				srv := dogwood_bench.NewDogwoodBenchmark(
					ctx,
					runtime.NewRandFromRoot(),
					outside,
					cfg,
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
			return cmdkit.Invoke(ctx, func() {

				srv := echo_bench.NewEchoBench(
					ctx,
					libnet.TCPNet{},
					benchDoSend,
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
	benchServerAddress string
	benchNumIterations int
	benchBytesPerPay   int
	benchNumCommits    int
	//
	benchStorageDir string
	benchInAddress  string
	benchOutAddress string
	benchTxsPerSec  int64
	//
	benchSCPDaemonPath  string
	benchSCPHandleAddr  string
	benchSCPControlAddr string
	benchSCPIsValidator bool
	//
	benchSchedPort int
	benchDebugPort int
	//
	benchDoSend          bool
	benchOurAddress      string
	benchTheirAddress    string
	benchNumConns        int
	benchMaxPendingCalls int
	benchNoEcho          bool
	//
	benchNumAccts    int
	benchTxsPerBlock int
	benchNumBlocks   int
	benchLoadgenAddr string
	benchSchedAddr   string
	benchNetworkHash string
	//
	benchMockScheduler bool
	//
	benchSkip int
)

func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.AddCommand(benchDogwoodCmd)
	benchDogwoodCmd.Flags().StringVar(&benchServerAddress, "addr", "localhost:9111", "address for broadcast api")
	benchDogwoodCmd.Flags().IntVar(&benchBytesPerPay, "bytes-per-pay", 300000, "bytes per payload")
	benchDogwoodCmd.Flags().IntVar(&benchNumIterations, "num-iterations", 10000, "number of iterations")

	scpdPath, _ := exec.LookPath("scpd")

	benchCmd.AddCommand(benchCoordinatorCmd)
	benchCoordinatorCmd.Flags().StringVar(&benchStorageDir, "storage-dir", "", "storage dir for consensus, scp, and catchup")
	benchCoordinatorCmd.Flags().StringVar(&benchInAddress, "in-addr", "localhost:7888", "bind address for internal apis")
	benchCoordinatorCmd.Flags().StringVar(&benchOutAddress, "out-addr", "localhost:9111", "bind address for external apis")
	benchCoordinatorCmd.Flags().Int64Var(&benchTxsPerSec, "txs-per-sec", 1e4, "transactions per second")
	benchCoordinatorCmd.Flags().IntVar(&benchSchedPort, "sched-port", 30555, "scheduler port")
	benchCoordinatorCmd.Flags().IntVar(&benchDebugPort, "debug-port", 40555, "debug port")
	benchCoordinatorCmd.Flags().StringVar(&benchSCPDaemonPath, "scpd-path", scpdPath, "scpd path")
	benchCoordinatorCmd.Flags().StringVar(&benchSCPHandleAddr, "scp-handle-addr", "localhost:3333", "scp handler addr")
	benchCoordinatorCmd.Flags().StringVar(&benchSCPControlAddr, "scp-control-addr", "localhost:3334", "scp controller addr")
	benchCoordinatorCmd.Flags().BoolVar(&benchSCPIsValidator, "scp-is-validator", true, "scp is validator")
	benchCoordinatorCmd.Flags().IntVar(&benchNumCommits, "num-commits", 100, "stop after number of commits")

	benchCmd.AddCommand(benchCoordinatorLoadgenCmd)
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchStorageDir, "storage-dir", "", "storage dir for consensus, scp, and catchup")
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchInAddress, "in-addr", "localhost:7888", "bind address for internal apis")
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchOutAddress, "out-addr", "localhost:9111", "bind address for external apis")
	benchCoordinatorLoadgenCmd.Flags().IntVar(&benchTxsPerBlock, "txs-per-block", 1e4, "transactions per block")
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchSchedAddr, "sched-addr", "", "scheduler address")
	benchCoordinatorLoadgenCmd.Flags().IntVar(&benchDebugPort, "debug-port", 40555, "debug port")
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchSCPDaemonPath, "scpd-path", scpdPath, "scpd path")
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchSCPHandleAddr, "scp-handle-addr", "localhost:3333", "scp handler addr")
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchSCPControlAddr, "scp-control-addr", "localhost:3334", "scp controller addr")
	benchCoordinatorLoadgenCmd.Flags().BoolVar(&benchSCPIsValidator, "scp-is-validator", true, "scp is validator")
	benchCoordinatorLoadgenCmd.Flags().IntVar(&benchNumAccts, "num-accts", 100, "number of accounts to generate")
	benchCoordinatorLoadgenCmd.Flags().IntVar(&benchNumBlocks, "num-blocks", 100, "number of blocks to generate")
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchLoadgenAddr, "loadgen-addr", "localhost:44400", "loadgen address")
	benchCoordinatorLoadgenCmd.Flags().StringVar(&benchNetworkHash, "network-hash", "11111111111111111111111111111112", "32-byte network hash in unprefixed base58")
	benchCoordinatorLoadgenCmd.Flags().BoolVar(&benchMockScheduler, "mock-scheduler", false, "use a mock scheduler")
	benchCoordinatorLoadgenCmd.Flags().IntVar(&benchSkip, "skip", 0, "number of initial blocks to skip reporting")

	benchCmd.AddCommand(benchEchoCmd)
	benchEchoCmd.Flags().BoolVar(&benchDoSend, "do-send", true, "send workload")
	benchEchoCmd.Flags().StringVar(&benchOurAddress, "our-addr", ":8181", "our address")
	benchEchoCmd.Flags().StringVar(&benchTheirAddress, "their-addr", ":8181", "their address")
	benchEchoCmd.Flags().IntVar(&benchNumConns, "num-conns", 10, "number of connections")
	benchEchoCmd.Flags().IntVar(&benchMaxPendingCalls, "max-pending-calls", 200, "max pending calls")
	benchEchoCmd.Flags().IntVar(&benchNumIterations, "num-iterations", 500e3, "number of iterations")
	benchEchoCmd.Flags().IntVar(&benchBytesPerPay, "bytes-per-pay", 60e3, "bytes per payload")
	benchEchoCmd.Flags().BoolVar(&benchNoEcho, "no-echo", true, "don't echo the request payload")
}
