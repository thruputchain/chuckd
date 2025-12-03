// main.go
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libruntime"
	"github.com/groundhogchain/chuckd/tools/hogsched"
	"github.com/groundhogchain/chuckd/tools/libtools"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
)

func main() {
	syscall.Setpgid(0, 0)
	libruntime.InstallKillProcessGroupOnInterrupt()
	liblog.LogLevel(zerolog.DebugLevel)

	ctx := context.Background()
	dot := libtools.DeriveDotHogtools(ctx)
	app := &cli.Command{
		Name:      "hog_sched",
		Usage:     "Schedule jobs",
		ArgsUsage: "<schedule_file>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "level",
				Aliases: []string{"l"},
				Value:   "info",
				Usage:   "log level",
			},
			&cli.StringFlag{
				Name:     "i",
				Aliases:  []string{"x"},
				Required: true,
				Usage:    "instance id",
			},
			&cli.StringFlag{
				Name:  "lockd",
				Usage: "address of hog lock daemon (overwrites HOG_LOCKD, which overwrites ~/.hogtools)",
				Value: dot.Lockd,
			},
			&cli.StringFlag{
				Name:  "job-dir",
				Usage: "run directory for jobs (overwrites HOG_JOB_DIR, which overwrites ~/.hogtools)",
				Value: dot.JobDir,
			},
			&cli.StringFlag{
				Name:  "dump-dir",
				Usage: "dump directory for jobs (overwrites HOG_DUMP_DIR, which overwrites ~/.hogtools)",
				Value: dot.DumpDir,
			},
			&cli.StringSliceFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "substitution key (repeated)",
			},
			&cli.StringSliceFlag{
				Name:    "value",
				Aliases: []string{"v"},
				Usage:   "substitution value (repeated, must match key)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {

			xid := c.String("i")
			level := c.String("level")
			lockdAddress := c.String("lockd")
			jobDir := c.String("job-dir")
			dumpDir := c.String("dump-dir")
			keys := c.StringSlice("key")
			values := c.StringSlice("value")

			// flag validation
			if len(keys) != len(values) {
				return errors.New("number of --key must match number of --value")
			}

			// log level
			zlevel, err := zerolog.ParseLevel(level)
			if err != nil {
				log.Fatal(err)
			}
			liblog.LogLevel(zlevel)
			libruntime.InstallStackTraceOnInterrupt()

			liblog.Infof("lockd=%v job_dir=%v dump_dir=%v", lockdAddress, jobDir, dumpDir)

			// spec argument
			if c.Args().Len() < 1 {
				_ = cli.ShowRootCommandHelp(c) // show usage for convenience
				return errors.New("missing required <schedule_file> argument")
			}
			specFile := c.Args().Get(0)

			// read spec
			specBody, err := os.ReadFile(specFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "reading %s: %v\n", specFile, err)
				return err
			}

			// rewrite spec with key/value substitutions
			kv := map[string]any{}
			for i := range keys {
				kv[keys[i]] = values[i]
			}

			// parse rewritten spec
			spec, err := hogsched.Parse(bytes.NewBuffer(specBody))
			if err != nil {
				fmt.Fprintf(os.Stderr, "parse %s: %v\n", specFile, err)
				return err
			}

			// schedule
			sched := hogsched.Scheduler{
				Net:          libnet.TCPNet{},
				LockdAddress: lockdAddress,
				JobDir:       jobDir,
				DumpDir:      dumpDir,
			}

			return sched.Schedule(ctx, xid, *spec, kv)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
