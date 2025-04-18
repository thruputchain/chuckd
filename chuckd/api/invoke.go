package api

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	xruntime "github.com/groundhogchain/chuckd/runtime"
)

func Invoke(f func()) error {

	ctx := context.Background()

	// mem profile
	defer func() {
		if memProfilePath != "" {
			f, err := os.Create(memProfilePath)
			if err != nil {
				log.Fatalf("could not create memory profile (%v)", err)
			}
			defer f.Close() // error handling omitted for example
			runtime.GC()    // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatalf("could not write memory profile (%v)", err)
			}
		}
	}()

	// cpu profile
	if cpuProfilePath != "" {
		f, err := os.Create(cpuProfilePath)
		if err != nil {
			log.Fatalf("could not create CPU profile (%v)", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("could not start CPU profile (%v)", err)
		}
		defer pprof.StopCPUProfile()
	}

	// start telemetry
	shutdown, err := xruntime.SetupTelemetry(ctx, instanceID)
	if err != nil {
		log.Fatalf("starting telemetry: %v", err)
		return err
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatalf("shutdown telemetry: %v", err)
		}
	}()

	//
	xerr := must.TryThru(f)
	if xerr != nil {
		if log.IsVerbose() {
			fmt.Fprintln(os.Stderr, xerr.StackString())
		}
		fmt.Fprintln(os.Stderr, xerr.Error())
		return xerr
	} else {
		return nil
	}
}

var cpuProfilePath string

func SetCPUProfilePath(filepath string) {
	cpuProfilePath = filepath
}

var memProfilePath string

func SetMemProfilePath(filepath string) {
	memProfilePath = filepath
}

var instanceID string

func SetInstanceID(id string) {
	instanceID = id
}
