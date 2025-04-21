package cmd

import (
	"github.com/groundhogchain/chuckd/bench"
	"github.com/groundhogchain/chuckd/chuckd/api"
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

	benchBroadcastCmd = &cobra.Command{
		Use:   "broadcast",
		Short: "start broadcast benchmark server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(func() {
				cfg := LoadConfig(ctx)
				srv := bench.NewBroadcastBenchmark(
					ctx,
					runtime.NewRandFromRoot(),
					*cfg,
					benchDeliveryPar,
					benchTreeFanOut,
					benchPaySize,
					benchPayPerChannel,
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
	benchDeliveryPar   int
	benchPaySize       int
	benchPayPerChannel int
)

func init() {
	rootCmd.AddCommand(benchCmd)

	benchCmd.AddCommand(benchBroadcastCmd)
	benchBroadcastCmd.Flags().StringVar(&benchServerAddress, "addr", "localhost:9111", "address for broadcast api")
	benchBroadcastCmd.Flags().IntVar(&benchTreeFanOut, "fanout", 5, "fanout of multicast trees")
	benchBroadcastCmd.Flags().IntVar(&benchDeliveryPar, "sendpar", 5, "deliveries in parallel")
	benchBroadcastCmd.Flags().IntVar(&benchPaySize, "paysize", 1000*1000*10, "payload size")
	benchBroadcastCmd.Flags().IntVar(&benchPayPerChannel, "payperchan", 100, "payloads per channel")
}
