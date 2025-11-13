// main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libruntime"
	"github.com/groundhogchain/chuckd/tools/hogsched"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
)

func main() {
	liblog.LogLevel(zerolog.DebugLevel)

	app := &cli.Command{
		Name:      "hog_sched",
		Usage:     "Schedule jobs defined in a spec file",
		ArgsUsage: "<spec_file>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "level",
				Value: "info",
				Usage: "log level",
			},
			&cli.StringFlag{
				Name:  "lockd",
				Usage: "address of hog lock daemon (also read from HOG_LOCKD)",
				Value: os.Getenv("HOG_LOCKD"),
			},
			&cli.StringFlag{
				Name:  "rundir",
				Usage: "run directory for jobs",
				Value: os.Getenv("HOG_RUNDIR"),
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {

			level := c.String("level")
			lockdAddress := c.String("lockd")
			runDir := c.String("rundir")

			// log level
			zlevel, err := zerolog.ParseLevel(level)
			if err != nil {
				log.Fatal(err)
			}
			liblog.LogLevel(zlevel)
			libruntime.StackTraceInterrupt()

			if c.Args().Len() < 1 {
				_ = cli.ShowRootCommandHelp(c) // show usage for convenience
				return errors.New("missing required <spec_file> argument")
			}
			specFile := c.Args().Get(0)

			f, err := os.Open(specFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "open %s: %v\n", specFile, err)
				return err
			}
			defer f.Close()

			spec, err := hogsched.Parse(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "parse %s: %v\n", specFile, err)
				return err
			}

			sched := hogsched.Config{
				Net:          libnet.TCPNet{},
				LockdAddress: lockdAddress,
				RunDir:       runDir,
			}

			return sched.Schedule(ctx, *spec)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
