package main

import (
	"context"
	"log"
	"os"

	"github.com/groundhogchain/chuckd/tools/libtools"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libruntime"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "hog_fwd",
		Usage: "forward github deploy keys to ssh agent when using git",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "level",
				Value: "info",
				Usage: "log level",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			level := c.String("level")

			// log level
			zlevel, err := zerolog.ParseLevel(level)
			if err != nil {
				log.Fatal(err)
			}
			liblog.LogLevel(zlevel)
			libruntime.InstallStackTraceOnInterrupt()

			return libtools.ForwardSshForGit(ctx)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
