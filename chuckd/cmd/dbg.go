package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/groundhogchain/chuckd/catchup"
	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/scp/solo/dial"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/spf13/cobra"
)

var (
	dbgCmd = &cobra.Command{
		Use:   "dbg",
		Short: "debugging tools",
		Long:  ``,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	dbgPingCmd = &cobra.Command{
		Use:   "ping",
		Short: "ping and xdr endpoint",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return api.Invoke(ctx, func() {
				err := PingService(ctx, dbgService, dbgFn, dbgAddress)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
				} else {
					fmt.Println("OK")
				}
			})
		},
	}
)

var (
	dbgAddress string
	dbgService string
	dbgFn      string
)

func init() {
	rootCmd.AddCommand(dbgCmd)

	dbgCmd.AddCommand(dbgPingCmd)
	dbgPingCmd.Flags().StringVar(&dbgAddress, "addr", "localhost:9111", "address")
	dbgPingCmd.Flags().StringVar(&dbgService, "svc", "", "service name")
	dbgPingCmd.Flags().StringVar(&dbgFn, "fn", "", "fn name")
}

func PingService(ctx context.Context, svc string, fn string, address string) error {
	switch svc {
	case "dogwood":
		return PingDogwood(ctx, address)
	case "catchup":
		return PingCatchup(ctx, address)
	case "coord":
		return PingCoordinator(ctx, address)
	case "discovery":
		return PingDiscovery(ctx, address)
	case "solofollower":
		switch fn {
		case "validate":
			return PingSoloFollowerValidate(ctx, address)
		default:
			return PingSoloFollower(ctx, address)
		}
	case "sololeader":
		return PingSoloLeader(ctx, address)
	default:
		return fmt.Errorf("unknown service")
	}
}

func PingDogwood(ctx context.Context, address string) error {
	c, err := dogwood.DialDogwoodClient(ctx, libnet.TCPNet{}, address)
	must.NoError(ctx, err)
	defer c.Close()
	return c.Dogwood_null(ctx)
}

func PingCatchup(ctx context.Context, address string) error {
	c, err := catchup.DialCatchupClient(ctx, libnet.TCPNet{}, address)
	must.NoError(ctx, err)
	defer c.Close()
	return c.Catchup_null(ctx)
}

func PingCoordinator(ctx context.Context, address string) error {
	c, err := coord.DialCoordinatorClient(ctx, libnet.TCPNet{}, address)
	must.NoError(ctx, err)
	defer c.Close()
	return c.Coordinator_null(ctx)
}

func PingDiscovery(ctx context.Context, address string) error {
	c, err := discovery.DialDiscoveryClient(ctx, libnet.TCPNet{}, address)
	must.NoError(ctx, err)
	defer c.Close()
	return c.Discovery_null(ctx)
}

func PingSoloFollower(ctx context.Context, address string) error {
	c, err := dial.DialSoloFollowerClient(ctx, libnet.TCPNet{}, address)
	must.NoError(ctx, err)
	defer c.Close()
	return c.Solofollower_null(ctx)
}

func PingSoloFollowerValidate(ctx context.Context, address string) error {
	c, err := dial.DialSoloFollowerClient(ctx, libnet.TCPNet{}, address)
	must.NoError(ctx, err)
	defer c.Close()
	_, err = c.Solofollower_validate(ctx, proto.SoloSignedValidate{})
	return err
}

func PingSoloLeader(ctx context.Context, address string) error {
	c, err := dial.DialSoloLeaderClient(ctx, libnet.TCPNet{}, address)
	must.NoError(ctx, err)
	defer c.Close()
	return c.Sololeader_null(ctx)
}
