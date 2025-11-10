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
		Name:  "hog_copy_from",
		Usage: "copy stuff from a hog to here",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "j",
				Usage:    "job id",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "from",
				Usage:    "from host",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:  "in",
				Usage: "input host/path (repeatable)",
			},
			&cli.StringSliceFlag{
				Name:  "out",
				Usage: "output host/path (repeatable, matches --in)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			job := c.String("j")
			from := c.String("from")
			ins := c.StringSlice("in")
			outs := c.StringSlice("out")

			if len(ins) != len(outs) {
				return fmt.Errorf("number of --in (%d) must match number of --out (%d)", len(ins), len(outs))
			}

			return libtools.HogCopyFrom(ctx, job, from, ins, outs)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
