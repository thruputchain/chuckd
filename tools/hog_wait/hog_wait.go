package main

import (
	"context"
	"log"
	"os"

	"github.com/groundhogchain/chuckd/tools/hoglock"
	"github.com/groundhogchain/chuckd/tools/libtools"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libruntime"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
)

func main() {
	liblog.LogLevel(zerolog.DebugLevel)

	ctx := context.Background()
	dot := libtools.DeriveDotHogtools(ctx)
	cmd := &cli.Command{
		Name:      "hog_wait",
		Usage:     "wait until there are raised signals with given prefixes (idempotent)",
		ArgsUsage: "<prefix>*",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "level",
				Value: "info",
				Usage: "log level",
			},
			&cli.StringFlag{
				Name:  "lockd",
				Usage: "address of hog lock daemon (overwrites HOG_LOCKD, which overwrites ~/.hogtools)",
				Value: dot.Lockd,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			level := c.String("level")
			lockdAddress := c.String("lockd")

			// log level
			zlevel, err := zerolog.ParseLevel(level)
			if err != nil {
				log.Fatal(err)
			}
			liblog.LogLevel(zlevel)
			libruntime.InstallStackTraceOnInterrupt()

			cli := hoglock.Client{
				Net:          libnet.TCPNet{},
				LockdAddress: lockdAddress,
			}

			ps := c.Args().Slice()

			xid := os.Getenv("HOG_XID")

			liblog.Infof("%v is waiting for signals with prefixes %v", xid, ps)
			return cli.Wait(ctx, xid, ps)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
