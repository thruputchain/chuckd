package cmd

import (
	"fmt"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/k8s"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/spf13/cobra"
)

var (
	k8sCmd = &cobra.Command{
		Use:   "k8s",
		Short: "build k8s configurations",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	k8sBenchBroadcastCmd = &cobra.Command{
		Use:   "bench-broadcast",
		Short: "generate a broadcast benchmark k8s config",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				fmt.Println(k8s.BenchBroadcastBlueprint(ctx, runtime.NewRandFromRoot(), k8sNumVs))
			})
		},
	}

	k8sBenchTestClusterCmd = &cobra.Command{
		Use:   "bench-testcluster",
		Short: "generate a testcluster benchmark k8s config",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				fmt.Println(k8s.BenchTestClusterBlueprint(ctx, runtime.NewRandFromRoot(), k8sNumVs, k8sTPS))
			})
		},
	}
)

var (
	k8sNumVs int
	k8sTPS   int
)

func init() {
	rootCmd.AddCommand(k8sCmd)

	k8sCmd.AddCommand(k8sBenchBroadcastCmd)
	k8sBenchBroadcastCmd.Flags().IntVar(&k8sNumVs, "numvs", 3, "number of validators")

	k8sCmd.AddCommand(k8sBenchTestClusterCmd)
	k8sBenchTestClusterCmd.Flags().IntVar(&k8sNumVs, "numvs", 3, "number of validators")
	k8sBenchTestClusterCmd.Flags().IntVar(&k8sTPS, "tps", 1e3, "transactions per second")
}
