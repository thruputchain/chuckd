package main

import (
	"context"
	"log"
	"os"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/tools/hoglock"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "hog_lockd",
		Usage: "hog cluster synchronization daemon",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "address",
				Value:    ":60006",
				Required: true,
				Usage:    "bind address",
			},
			&cli.StringSliceFlag{
				Name:  "hosts",
				Usage: "available hosts (repeatable)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			addr := c.String("address")
			_ = c.StringSlice("hosts")

			hoglock.NewService().Start(ctx, libnet.TCPNet{}, addr)
			select {}
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
