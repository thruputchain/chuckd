package cmd

import (
	"github.com/groundhogchain/chuckd/coord"
	"github.com/spf13/cobra"
)

var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "coordinator server control",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	serverStartCmd = &cobra.Command{
		Use:   "start",
		Short: "start coordinator server",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			coord.StartServer(ctx, coordAddress)
			<-(chan int)(nil) // hang
		},
	}
)

var (
	coordAddress string
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.AddCommand(serverStartCmd)
	serverStartCmd.Flags().StringVar(&coordAddress, "caddr", "localhost:7888", "coordinator api address")
}
