package cmd

import (
	"fmt"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/deploy/k8s"
	"github.com/groundhogchain/chuckd/deploy/metal"
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

	deployK8sBenchBroadcastCmd = &cobra.Command{
		Use:   "k8s-bench-broadcast",
		Short: "generate a broadcast benchmark k8s config",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				fmt.Println(k8s.BenchBroadcastBlueprint(ctx, runtime.NewRandFromRoot(), deployNumVs))
			})
		},
	}

	deployK8sBenchTestClusterCmd = &cobra.Command{
		Use:   "k8s-bench-testcluster",
		Short: "generate a testcluster benchmark k8s config",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				fmt.Println(k8s.BenchTestClusterBlueprint(ctx, runtime.NewRandFromRoot(), deployNumVs, deployTPS))
			})
		},
	}

	deployLocalBenchTestClusterCmd = &cobra.Command{
		Use:   "local-bench-testcluster",
		Short: "generate a local testcluster benchmark deploy shell script",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				fmt.Println(metal.BenchTestClusterShell(ctx, runtime.NewRandFromRoot(), deployNumVs, deployTPS))
			})
		},
	}
)

var (
	deployNumVs int
	deployTPS   int
)

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.AddCommand(deployK8sBenchBroadcastCmd)
	deployK8sBenchBroadcastCmd.Flags().IntVar(&deployNumVs, "num-vs", 3, "number of validators")

	deployCmd.AddCommand(deployK8sBenchTestClusterCmd)
	deployK8sBenchTestClusterCmd.Flags().IntVar(&deployNumVs, "num-vs", 3, "number of validators")
	deployK8sBenchTestClusterCmd.Flags().IntVar(&deployTPS, "tps", 1e3, "transactions per second")

	deployCmd.AddCommand(deployLocalBenchTestClusterCmd)
	deployLocalBenchTestClusterCmd.Flags().IntVar(&deployNumVs, "num-vs", 3, "number of validators")
	deployLocalBenchTestClusterCmd.Flags().IntVar(&deployTPS, "tps", 1e3, "transactions per second")
}
