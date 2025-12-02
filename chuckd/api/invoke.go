package api

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libruntime"
	"github.com/groundhogchain/chuckd/lib/must"
	xruntime "github.com/groundhogchain/chuckd/runtime"
)

func flushMemProfile(memProfilePath string) {
	if memProfilePath == "" {
		return
	}
	f, err := os.Create(memProfilePath)
	if err != nil {
		liblog.Fatalf("could not create memory profile (%v)", err)
	}
	defer f.Close()
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		liblog.Fatalf("could not write memory profile (%v)", err)
	}
}

func flushMutexProfile(mutexProfilePath string) {
	if mutexProfilePath == "" {
		return
	}
	f, err := os.Create(mutexProfilePath)
	if err != nil {
		liblog.Fatalf("could not create mutex profile (%v)", err)
	}
	defer f.Close()
	if err := pprof.Lookup("mutex").WriteTo(f, 0); err != nil {
		liblog.Fatalf("could not write mutex profile (%v)", err)
	}
}

func flushBlockProfile(blockProfilePath string) {
	if blockProfilePath == "" {
		return
	}
	f, err := os.Create(blockProfilePath)
	if err != nil {
		liblog.Fatalf("could not create block profile (%v)", err)
	}
	defer f.Close()
	if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
		liblog.Fatalf("could not write block profile (%v)", err)
	}
}

func Invoke(ctx context.Context, f func()) error {

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
			liblog.Fatalf("could not create CPU profile (%v)", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			liblog.Fatalf("could not start CPU profile (%v)", err)
		}
		defer pprof.StopCPUProfile()
		hooks = append(hooks,
			func() {
				pprof.StopCPUProfile()
				f.Close()
			},
		)
	}

	// trace profile
	if traceProfilePath != "" {
		f, err := os.Create(traceProfilePath)
		if err != nil {
			liblog.Fatalf("could not create trace profile (%v)", err)
		}
		defer f.Close()
		if err := trace.Start(f); err != nil {
			liblog.Fatalf("could not start trace profile (%v)", err)
		}
		defer trace.Stop()
		hooks = append(hooks,
			func() {
				trace.Stop()
				f.Close()
			},
		)
	}

	// mutex profile
	if mutexProfilePath != "" {
		runtime.SetMutexProfileFraction(1)
		defer flushMutexProfile(mutexProfilePath)
		hooks = append(hooks, func() { flushMutexProfile(mutexProfilePath) })
	}

	// block profile
	if blockProfilePath != "" {
		runtime.SetBlockProfileRate(1)
		defer flushBlockProfile(blockProfilePath)
		hooks = append(hooks, func() { flushBlockProfile(blockProfilePath) })
	}

	if stackTrace {
		libruntime.InstallStackTraceOnInterrupt()
	}
	for _, hook := range hooks {
		libruntime.InstallInterruptHook(hook)
	}

	// start telemetry
	shutdown, err := xruntime.SetupTelemetry(ctx, instanceID, metricsAddr)
	if err != nil {
		liblog.Fatalf("starting telemetry: %v", err)
		return err
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			liblog.Fatalf("shutdown telemetry: %v", err)
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
		if stackTrace {
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

var mutexProfilePath string

func SetMutexProfilePath(filepath string) {
	mutexProfilePath = filepath
}

var blockProfilePath string

func SetBlockProfilePath(filepath string) {
	blockProfilePath = filepath
}

var traceProfilePath string

func SetTraceProfilePath(filepath string) {
	traceProfilePath = filepath
}

var instanceID string

func SetInstanceID(id string) {
	instanceID = id
}

var metricsAddr string

func SetMetricsAddr(addr string) {
	metricsAddr = addr
}

var stackTrace bool

func SetStackTrace(s bool) {
	stackTrace = s
}
