package cmd

import (
	"fmt"
	"os/exec"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libmetrics"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/system/coordinator_bench"
	"github.com/groundhogchain/chuckd/system/dogwood_bench"
	"github.com/groundhogchain/chuckd/system/echo_bench"
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

	benchCoordinatorCmd = &cobra.Command{
		Use:   "coordinator",
		Short: "start coordinator benchmark",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				ctx = libmetrics.WithMetricAttributes(ctx,
					attribute.String("cmd", "bench coordinator"),
					attribute.Int64("txs_per_sec", benchTxPerSec),
				)
				cfg := LoadCoordinatorConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				outside := libnet.Network(libnet.TCPNet{})
				testCfg := coordinator_bench.CoordinatorBenchConfig{
					TxsPerSec: benchTxPerSec,
					SchedPort: benchSchedPort,
				}
				sys := coordinator_bench.NewCoordinatorBenchSystem(
					ctx,
					libtime.RealTime{},
					runtime.NewRandFromRoot(),
					outside,
					cfg,
					testCfg,
				)
				opcfg := coordinator.OperationalConfig{
					StorageDir:     benchStorageDir,
					SchedAddr:      fmt.Sprintf("localhost:%d", benchSchedPort),
					DebugPort:      benchDebugPort,
					InAddr:         benchInAddress,
					OutAddr:        benchOutAddress,
					WalletAddr:     benchWalletAddress,
					SCPDaemonPath:  benchSCPDaemonPath,
					SCPHandleAddr:  benchSCPHandleAddr,
					SCPControlAddr: benchSCPControlAddr,
					IsValidator:    benchSCPIsValidator,
				}
				ready := make(chan struct{})
				sys.Start(ctx, opcfg, benchNumCommits, ready)
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
			return api.Invoke(ctx, func() {
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
			return api.Invoke(ctx, func() {

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
	benchStorageDir    string
	benchInAddress     string
	benchOutAddress    string
	benchWalletAddress string
	benchTxPerSec      int64
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
	benchCoordinatorCmd.Flags().StringVar(&benchWalletAddress, "wallet-addr", "localhost:6666", "bind address for wallet api")
	benchCoordinatorCmd.Flags().Int64Var(&benchTxPerSec, "tx-per-sec", 1e4, "transactions per second")
	benchCoordinatorCmd.Flags().IntVar(&benchSchedPort, "sched-port", 30555, "scheduler port")
	benchCoordinatorCmd.Flags().IntVar(&benchDebugPort, "debug-port", 40555, "debug port")
	benchCoordinatorCmd.Flags().StringVar(&benchSCPDaemonPath, "scpd-path", scpdPath, "scpd path")
	benchCoordinatorCmd.Flags().StringVar(&benchSCPHandleAddr, "scp-handle-addr", "localhost:3333", "scp handler addr")
	benchCoordinatorCmd.Flags().StringVar(&benchSCPControlAddr, "scp-control-addr", "localhost:3334", "scp controller addr")
	benchCoordinatorCmd.Flags().BoolVar(&benchSCPIsValidator, "scp-is-validator", true, "scp is validator")
	benchCoordinatorCmd.Flags().IntVar(&benchNumCommits, "num-commits", 100, "stop after number of commits")

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
