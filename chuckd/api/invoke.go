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

func flushMemProfile(memProfilePath string) {
	if memProfilePath == "" {
		return
	}
	f, err := os.Create(memProfilePath)
	if err != nil {
		log.Fatalf("could not create memory profile (%v)", err)
	}
	defer f.Close()
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatalf("could not write memory profile (%v)", err)
	}
}

func Invoke(f func()) error {

	ctx := context.Background()

	hooks := []func(){}

	// mem profile
	if memProfilePath != "" {
		defer flushMemProfile(memProfilePath)
		hooks = append(hooks, func() { flushMemProfile(memProfilePath) })
	}

	// cpu profile
	if cpuProfilePath != "" {
		f, err := os.Create(cpuProfilePath)
		if err != nil {
			log.Fatalf("could not create CPU profile (%v)", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("could not start CPU profile (%v)", err)
		}
		defer pprof.StopCPUProfile()
		hooks = append(hooks,
			func() {
				pprof.StopCPUProfile()
				f.Close()
			},
		)
	}

	xruntime.CaptureInterrupts(hooks...)

	// start telemetry
	shutdown, err := xruntime.SetupTelemetry(ctx, instanceID, metricsAddr)
	if err != nil {
		log.Fatalf("starting telemetry: %v", err)
		return err
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatalf("shutdown telemetry: %v", err)
		}
	}()
	hooks = append(hooks,
		func() {
			shutdown(ctx)
		},
	)

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

var metricsAddr string

func SetMetricsAddr(addr string) {
	metricsAddr = addr
}
