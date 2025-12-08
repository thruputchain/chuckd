package main

import (
	"context"
	"log"
	"os"

	"github.com/groundhogchain/chuckd/tools/hoglock"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libruntime"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "hog_lockd",
		Usage: "hog cluster synchronization daemon",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "level",
				Value: "info",
				Usage: "log level",
			},
			&cli.StringFlag{
				Name:  "address",
				Value: ":60006",
				// Required: true,
				Usage: "bind address",
			},
			&cli.StringSliceFlag{
				Name:  "host",
				Usage: "available hosts (repeatable)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			level := c.String("level")
			addr := c.String("address")
			host := c.StringSlice("host")

			// log level
			zlevel, err := zerolog.ParseLevel(level)
			if err != nil {
				log.Fatal(err)
			}
			liblog.LogLevel(zlevel)
			libruntime.InstallStackTraceOnInterrupt()

			hoglock.NewService(host).Start(ctx, libnet.TCPNet{}, addr)
			select {}
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
