package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/groundhogchain/chuckd/tools/libtools"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "hog_exec",
		Usage: "execute a script on a hog",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "j",
				Usage:    "job id",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "host",
				Usage: "host to run on",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			job := c.String("j")
			host := c.String("host")

			fmt.Fprintf(os.Stdout, "executing for %s on %v\n", job, host)
			_, _, err := libtools.HogExecReader(ctx, job, host, os.Stdin)
			return err
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
