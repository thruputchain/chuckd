package cmd

import (
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/deploy/k8s"
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

	deployK8sBenchDogwoodCmd = &cobra.Command{
		Use:   "k8s-bench-dogwood",
		Short: "generate a dogwood benchmark k8s config",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				fmt.Println(k8s.BenchDogwoodBlueprint(ctx, runtime.NewRandFromRoot(), deployNumVs, deployFanOut))
			})
		},
	}

	deployK8sBenchTestClusterCmd = &cobra.Command{
		Use:   "k8s-bench-testcluster",
		Short: "generate a testcluster benchmark k8s config",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				fmt.Println(k8s.BenchTestClusterBlueprint(ctx, runtime.NewRandFromRoot(),
					time.Second, deployNumVs, deployFanOut, deployTPS))
			})
		},
	}

	deployLocalBenchTestClusterCmd = &cobra.Command{
		Use:   "local-coordinator-bench",
		Short: "generate a local coordinator benchmark deploy shell script",
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
						deployTPS,
					))
			})
		},
	}
)

var (
	deployFanOut    int
	deployNumVs     int
	deployTPS       int
	deployRoundMsec int
)

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.AddCommand(deployK8sBenchDogwoodCmd)
	deployK8sBenchDogwoodCmd.Flags().IntVar(&deployFanOut, "fan-out", 5, "tree fan out")
	deployK8sBenchDogwoodCmd.Flags().IntVar(&deployNumVs, "num-vs", 3, "number of validators")

	deployCmd.AddCommand(deployK8sBenchTestClusterCmd)
	deployK8sBenchTestClusterCmd.Flags().IntVar(&deployFanOut, "fan-out", 5, "tree fan out")
	deployK8sBenchTestClusterCmd.Flags().IntVar(&deployNumVs, "num-vs", 3, "number of validators")
	deployK8sBenchTestClusterCmd.Flags().IntVar(&deployTPS, "tx-per-sec", 1e3, "transactions per second")

	deployCmd.AddCommand(deployLocalBenchTestClusterCmd)
	deployLocalBenchTestClusterCmd.Flags().IntVar(&deployFanOut, "fan-out", 5, "tree fan out")
	deployLocalBenchTestClusterCmd.Flags().IntVar(&deployNumVs, "num-vs", 3, "number of validators")
	deployLocalBenchTestClusterCmd.Flags().IntVar(&deployTPS, "tx-per-sec", 1e3, "transactions per second")
	deployLocalBenchTestClusterCmd.Flags().IntVar(&deployRoundMsec, "round-msec", 3000, "target round duration, msec")
}
