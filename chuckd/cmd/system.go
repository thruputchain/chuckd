package cmd

import (
	"fmt"
	"os/exec"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/system/dogwood"
	"github.com/spf13/cobra"
)

var (
	sysCmd = &cobra.Command{
		Use:   "sys",
		Short: "systems control",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	sysCoordinatorCmd = &cobra.Command{
		Use:   "coordinator",
		Short: "start coordinator system",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				cfg := LoadCoordinatorConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				inside := libnet.Network(libnet.TCPNet{})
				outside := libnet.Network(libnet.TCPNet{})
				srv := coordinator.NewSystem(
					ctx,
					nil,
					runtime.NewRandFromRoot(),
					cfg,
					inside,
					outside,
				)
				opcfg := coordinator.OperationalConfig{
					StorageDir:     serverStorageDir,
					SchedAddr:      serverSchedAddress,
					DebugPort:      0, // not used by coordinator system
					InAddr:         serverInAddress,
					OutAddr:        serverOutAddress,
					WalletAddr:     serverWalletAddress,
					SCPDaemonPath:  serverSCPDaemonPath,
					SCPHandleAddr:  serverSCPHandleAddr,
					SCPControlAddr: serverSCPControlAddr,
					IsValidator:    serverSCPIsValidator,
				}
				srv.Start(ctx, opcfg)
				select {} // hang
			})
		},
	}

	sysDogwoodCmd = &cobra.Command{
		Use:   "dogwood",
		Short: "start dogwood (standalone disseminator) system",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				cfg := LoadDogwoodConfig(ctx)
				srv := dogwood.NewSystem(
					ctx,
					runtime.NewRandFromRoot(),
					libnet.Network(libnet.TCPNet{}),
					cfg,
				)
				srv.Start(ctx, serverOutAddress, serverInAddress)
				fmt.Println("dogwood is ready")
				select {} // hang
			})
		},
	}
)

var (
	serverStorageDir    string
	serverSchedAddress  string
	serverInAddress     string
	serverOutAddress    string
	serverWalletAddress string
	//
	serverSCPDaemonPath  string
	serverSCPHandleAddr  string
	serverSCPControlAddr string
	serverSCPIsValidator bool
)

func init() {
	rootCmd.AddCommand(sysCmd)

	scpdPath, _ := exec.LookPath("scpd")

	sysCmd.AddCommand(sysCoordinatorCmd)
	sysCoordinatorCmd.Flags().StringVar(&serverStorageDir, "storage-dir", "", "storage dir for consensus, scp, and catchup")
	sysCoordinatorCmd.Flags().StringVar(&serverSchedAddress, "sched-addr", "localhost:7555", "address of scheduler")
	sysCoordinatorCmd.Flags().StringVar(&serverInAddress, "in-addr", "localhost:7888", "bind address for internal apis")
	sysCoordinatorCmd.Flags().StringVar(&serverOutAddress, "out-addr", "localhost:9111", "bind address for external apis")
	sysCoordinatorCmd.Flags().StringVar(&serverWalletAddress, "wallet-addr", "localhost:6666", "bind address for wallet api")
	sysCoordinatorCmd.Flags().StringVar(&serverSCPDaemonPath, "scpd-path", scpdPath, "scpd path")
	sysCoordinatorCmd.Flags().StringVar(&serverSCPHandleAddr, "scp-handle-addr", "localhost:3333", "scp handler addr")
	sysCoordinatorCmd.Flags().StringVar(&serverSCPControlAddr, "scp-control-addr", "localhost:3334", "scp controller addr")
	sysCoordinatorCmd.Flags().BoolVar(&serverSCPIsValidator, "scp-is-validator", true, "scp is validator")

	sysCmd.AddCommand(sysDogwoodCmd)
	sysDogwoodCmd.Flags().StringVar(&serverInAddress, "in-addr", "localhost:10101", "address for api")
	sysDogwoodCmd.Flags().StringVar(&serverOutAddress, "out-addr", ":33055", "address for peers")
}
