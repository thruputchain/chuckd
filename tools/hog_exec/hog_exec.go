package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/groundhogchain/chuckd/lib/must"
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

			buf, err := io.ReadAll(os.Stdin)
			must.NoError(ctx, err)
			script := string(buf)

			fmt.Fprintf(os.Stdout, "executing for %s on %v\n", job, host)
			_, _, err = libtools.HogExec(ctx, job, host, script)
			return err
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
