package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/groundhogchain/chuckd/chuckd/cmdkit"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/system/dogwood"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
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
			return cmdkit.Invoke(ctx, func() {
				cfg := LoadCoordinatorConfig(ctx)
				must.Assertf(ctx, cfg.Validator_config != nil, "missing validator config section")
				var notify coordinator.Notify
				var hostedBootstrapNotify *coordinator.NotifyReceive
				if serverHostedBootstrapMetadataPath != "" {
					hostedBootstrapNotify = coordinator.NewNotifyReceive()
					notify = hostedBootstrapNotify
				}
				inside := libnet.Network(libnet.TCPReuseNet{})
				outside := libnet.Network(libnet.TCPReuseNet{})
				srv := coordinator.NewSystem(
					ctx,
					notify,
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
					SCPDaemonPath:  serverSCPDaemonPath,
					SCPHandleAddr:  serverSCPHandleAddr,
					SCPControlAddr: serverSCPControlAddr,
					IsValidator:    serverSCPIsValidator,
				}
				ctl := srv.Start(ctx, opcfg)
				if serverGenesisPath != "" {
					loadGenesisIfNeeded(ctx, ctl, serverGenesisPath)
				}
				if serverHostedBootstrapMetadataPath != "" {
					must.NoError(ctx, initializeHostedBootstrapWithCoordinator(ctx, hostedBootstrapConfig{
						metadataPath:       serverHostedBootstrapMetadataPath,
						schedulerAddress:   serverSchedAddress,
						indexerAddress:     serverHostedBootstrapIndexerAddress,
						dbGatewayAddress:   serverHostedBootstrapDbGatewayAddress,
						viewGatewayAddress: serverHostedBootstrapViewGatewayAddress,
						publicRPCURL:       serverHostedBootstrapPublicRPCURL,
						runtimeManifestOut: serverHostedBootstrapRuntimeManifestOut,
					}, ctl, hostedBootstrapNotify))
				} else {
					ctl.SignalGenesisIsReady()
				}
				select {} // hang
			})
		},
	}

	sysDogwoodCmd = &cobra.Command{
		Use:   "dogwood",
		Short: "start dogwood (standalone disseminator) system",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdkit.Invoke(ctx, func() {
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
	serverStorageDir   string
	serverSchedAddress string
	serverInAddress    string
	serverOutAddress   string
	//
	serverSCPDaemonPath  string
	serverSCPHandleAddr  string
	serverSCPControlAddr string
	serverSCPIsValidator bool
	serverGenesisPath    string

	serverHostedBootstrapMetadataPath       string
	serverHostedBootstrapIndexerAddress     string
	serverHostedBootstrapDbGatewayAddress   string
	serverHostedBootstrapViewGatewayAddress string
	serverHostedBootstrapPublicRPCURL       string
	serverHostedBootstrapRuntimeManifestOut string
)

func loadGenesisConfigurationFile(ctx context.Context, filePath string) proto.GenesisConfiguration {
	data, err := os.ReadFile(filePath)
	must.Assertf(ctx, err == nil, "reading genesis file (%w)", err)

	var genesis proto.GenesisConfiguration
	err = libxdr.JsonToXdr(&genesis, data)
	must.NoError(ctx, err)
	return genesis
}

type genesisControl interface {
	GetTip(context.Context) (proto.SchedulerGetTipResult, error)
	LoadGenesisConfiguration(context.Context, proto.GenesisConfiguration) (proto.DbSchedResult, error)
}

func loadGenesisIfNeeded(ctx context.Context, ctl genesisControl, genesisPath string) {
	tip, err := ctl.GetTip(ctx)
	must.NoError(ctx, err)

	switch tip.Outcome {
	case proto.SchedulerGetTipOK:
		liblog.Infof(ctx, "scheduler genesis already loaded at %v", protoutil.DbTipString(*tip.Db_tip()))
		return
	case proto.SchedulerGetTipNoGenesisBlock:
		genesis := loadGenesisConfigurationFile(ctx, genesisPath)
		result, err := ctl.LoadGenesisConfiguration(ctx, genesis)
		must.NoError(ctx, err)
		if result != proto.DbSchedResultOK {
			tip, err := ctl.GetTip(ctx)
			must.NoError(ctx, err)
			if tip.Outcome == proto.SchedulerGetTipOK {
				liblog.Infof(ctx, "scheduler genesis loaded concurrently at %v", protoutil.DbTipString(*tip.Db_tip()))
				return
			}
		}
		must.Assertf(ctx, result == proto.DbSchedResultOK, "loading genesis failed with %v", result)
		liblog.Infof(ctx, "loaded genesis configuration from %s", genesisPath)
		return
	default:
		must.Errorf(ctx, "unexpected scheduler tip outcome %v", tip.Outcome)
	}
}

func init() {
	rootCmd.AddCommand(sysCmd)

	scpdPath, _ := exec.LookPath("scpd")

	sysCmd.AddCommand(sysCoordinatorCmd)
	sysCoordinatorCmd.Flags().StringVar(&serverStorageDir, "storage-dir", "", "storage dir for consensus, scp, and catchup")
	sysCoordinatorCmd.Flags().StringVar(&serverSchedAddress, "sched-addr", "localhost:7555", "address of scheduler")
	sysCoordinatorCmd.Flags().StringVar(&serverInAddress, "in-addr", "localhost:7888", "bind address for internal apis")
	sysCoordinatorCmd.Flags().StringVar(&serverOutAddress, "out-addr", "localhost:9111", "bind address for external apis")
	sysCoordinatorCmd.Flags().StringVar(&serverSCPDaemonPath, "scpd-path", scpdPath, "scpd path")
	sysCoordinatorCmd.Flags().StringVar(&serverSCPHandleAddr, "scp-handle-addr", "localhost:3333", "scp handler addr")
	sysCoordinatorCmd.Flags().StringVar(&serverSCPControlAddr, "scp-control-addr", "localhost:3334", "scp controller addr")
	sysCoordinatorCmd.Flags().BoolVar(&serverSCPIsValidator, "scp-is-validator", true, "scp is validator")
	sysCoordinatorCmd.Flags().StringVar(&serverGenesisPath, "genesis-file", "", "JSON-XDR genesis configuration to load when scheduler has no genesis")
	sysCoordinatorCmd.Flags().StringVar(&serverHostedBootstrapMetadataPath, "hosted-bootstrap-metadata", "", "hosted bootstrap metadata generated by hog-hosted-bootstrap; initializes before Baton starts")
	sysCoordinatorCmd.Flags().StringVar(&serverHostedBootstrapIndexerAddress, "hosted-bootstrap-indexer", "127.0.0.1:8061", "indexer address for hosted runtime manifest")
	sysCoordinatorCmd.Flags().StringVar(&serverHostedBootstrapDbGatewayAddress, "hosted-bootstrap-db-gateway", "", "DbGateway address for hosted runtime manifest")
	sysCoordinatorCmd.Flags().StringVar(&serverHostedBootstrapViewGatewayAddress, "hosted-bootstrap-view-gateway", "", "ViewGateway address for hosted runtime manifest")
	sysCoordinatorCmd.Flags().StringVar(&serverHostedBootstrapPublicRPCURL, "hosted-bootstrap-public-rpc-url", "", "public RPC URL for hosted runtime manifest")
	sysCoordinatorCmd.Flags().StringVar(&serverHostedBootstrapRuntimeManifestOut, "hosted-bootstrap-runtime-manifest-out", "", "write hosted runtime manifest after bootstrap commit")

	sysCmd.AddCommand(sysDogwoodCmd)
	sysDogwoodCmd.Flags().StringVar(&serverInAddress, "in-addr", "localhost:10101", "address for api")
	sysDogwoodCmd.Flags().StringVar(&serverOutAddress, "out-addr", ":33055", "address for peers")
}
