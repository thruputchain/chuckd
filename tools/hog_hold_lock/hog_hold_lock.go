package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/groundhogchain/chuckd/tools/libtools"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "hog_hold_lock",
		Usage: "acquire and hold locks on multiple hogs",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "j",
				Usage:    "job id",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:  "host",
				Usage: "host to lock (repeatable)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			jobID := c.String("j")
			hosts := c.StringSlice("host")

			for i := range hosts {
				hosts[i] = strings.TrimSpace(hosts[i])
			}

			fmt.Fprintf(os.Stdout, "acquiring locks for %s on %v\n", jobID, hosts)
			return libtools.SyncHogLock.HoldLockMany(ctx, jobID, hosts)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
