package cmd

import (
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/deploy/script"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/spf13/cobra"
)

var (
	deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "tools to build deploy configurations",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	deployCoordinatorBenchLocalCmd = &cobra.Command{
		Use:   "coordinator-bench-local",
		Short: "generate a local coordinator benchmark deploy script",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				fmt.Println(
					script.CoordinatorBenchShellLocal(
						ctx,
						runtime.NewRandFromRoot(),
						time.Millisecond*time.Duration(deployRoundMsec),
						deployNumVs,
						deployFanOut,
						deployTxsPerSec,
					))
			})
		},
	}

	deployCoordinatorBenchNetCmd = &cobra.Command{
		Use:   "coordinator-bench-net",
		Short: "generate multi-node coordinator benchmark deploy scripts",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				bcfg := script.CoordinatorBenchConfig{
					ValidatorPort:  deployValidatorPort,
					WalletPort:     deployWalletPort,
					MetricsPort:    deployMetricsPort,
					DebugPort:      deployDebugPort,
					ScpHandlePort:  deployScpHandlePort,
					ScpControlPort: deployScpControlPort,
				}
				scripts := script.CoordinatorBenchScripts(
					ctx,
					runtime.NewRandFromRoot(),
					bcfg,
					deployHosts,
					time.Millisecond*time.Duration(deployRoundMsec),
					deployFanOut,
					deployTxsPerSec,
					deployNumCommits,
				)
				for i, host := range deployHosts {
					fmt.Printf("### launch-coordinator-bench-%s.sh\n", host)
					fmt.Println(scripts[i])
				}
			})
		},
	}
)

var (
	deployNumVs int
	//
	deployFanOut     int
	deployTxsPerSec  int
	deployNumCommits int
	deployRoundMsec  int
	//
	deployHosts          []string
	deployValidatorPort  int
	deployWalletPort     int
	deployMetricsPort    int
	deployDebugPort      int
	deployScpHandlePort  int
	deployScpControlPort int
)

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.AddCommand(deployCoordinatorBenchLocalCmd)
	deployCoordinatorBenchLocalCmd.Flags().IntVar(&deployFanOut, "fan-out", 5, "tree fan out")
	deployCoordinatorBenchLocalCmd.Flags().IntVar(&deployNumVs, "num-vs", 3, "number of validators")
	deployCoordinatorBenchLocalCmd.Flags().IntVar(&deployTxsPerSec, "tx-per-sec", 1e3, "transactions per second")
	deployCoordinatorBenchLocalCmd.Flags().IntVar(&deployRoundMsec, "round-msec", 3000, "target round duration, msec")

	deployCmd.AddCommand(deployCoordinatorBenchNetCmd)
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployFanOut, "fan-out", 5, "tree fan out")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployTxsPerSec, "tx-per-sec", 1e3, "transactions per second")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployNumCommits, "num-commits", 100, "stop after number of commits")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployRoundMsec, "round-msec", 3000, "target round duration, msec")
	//
	deployCoordinatorBenchNetCmd.Flags().StringSliceVar(&deployHosts, "hosts", []string{"hog6", "hog7", "hog8"}, "hosts")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployValidatorPort, "v-port", 50000, "validator port")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployWalletPort, "w-port", 40000, "wallet port")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployMetricsPort, "m-port", 2223, "metrics port")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployDebugPort, "d-port", 5500, "debug port")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployScpHandlePort, "scp-handle-port", 3300, "scp handle port")
	deployCoordinatorBenchNetCmd.Flags().IntVar(&deployScpControlPort, "scp-control-port", 4400, "scp control port")
}
