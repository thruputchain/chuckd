package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thruputchain/chuckd"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "display version info",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(chuckd.Short())
		},
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}
