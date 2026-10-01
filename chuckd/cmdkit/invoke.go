package cmdkit

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	xruntime "github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libruntime"
	"github.com/groundhogchain/lib-go/must"
)

func flushMemProfile(ctx context.Context, memProfilePath string) {
	if memProfilePath == "" {
		return
	}
	f, err := os.Create(memProfilePath)
	if err != nil {
		liblog.Fatalf(ctx, "could not create memory profile (%v)", err)
	}
	defer f.Close()
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		liblog.Fatalf(ctx, "could not write memory profile (%v)", err)
	}
}

func flushMutexProfile(ctx context.Context, mutexProfilePath string) {
	if mutexProfilePath == "" {
		return
	}
	f, err := os.Create(mutexProfilePath)
	if err != nil {
		liblog.Fatalf(ctx, "could not create mutex profile (%v)", err)
	}
	defer f.Close()
	if err := pprof.Lookup("mutex").WriteTo(f, 0); err != nil {
		liblog.Fatalf(ctx, "could not write mutex profile (%v)", err)
	}
}

func flushBlockProfile(ctx context.Context, blockProfilePath string) {
	if blockProfilePath == "" {
		return
	}
	f, err := os.Create(blockProfilePath)
	if err != nil {
		liblog.Fatalf(ctx, "could not create block profile (%v)", err)
	}
	defer f.Close()
	if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
		liblog.Fatalf(ctx, "could not write block profile (%v)", err)
	}
}

func Invoke(ctx context.Context, f func()) error {

	hooks := []func(os.Signal){}
	applyPprofRates(runtime.SetMutexProfileFraction, runtime.SetBlockProfileRate)

	// mem profile
	if memProfilePath != "" {
		defer flushMemProfile(ctx, memProfilePath)
		hooks = append(hooks, func(os.Signal) { flushMemProfile(ctx, memProfilePath) })
	}

	// cpu profile
	if cpuProfilePath != "" {
		f, err := os.Create(cpuProfilePath)
		if err != nil {
			liblog.Fatalf(ctx, "could not create CPU profile (%v)", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			liblog.Fatalf(ctx, "could not start CPU profile (%v)", err)
		}
		defer pprof.StopCPUProfile()
		hooks = append(hooks,
			func(os.Signal) {
				pprof.StopCPUProfile()
				f.Close()
			},
		)
	}

	// trace profile
	if traceProfilePath != "" {
		f, err := os.Create(traceProfilePath)
		if err != nil {
			liblog.Fatalf(ctx, "could not create trace profile (%v)", err)
		}
		defer f.Close()
		if err := trace.Start(f); err != nil {
			liblog.Fatalf(ctx, "could not start trace profile (%v)", err)
		}
		defer trace.Stop()
		hooks = append(hooks,
			func(os.Signal) {
				trace.Stop()
				f.Close()
			},
		)
	}

	// mutex profile
	if mutexProfilePath != "" {
		runtime.SetMutexProfileFraction(1)
		defer flushMutexProfile(ctx, mutexProfilePath)
		hooks = append(hooks, func(os.Signal) { flushMutexProfile(ctx, mutexProfilePath) })
	}

	// block profile
	if blockProfilePath != "" {
		runtime.SetBlockProfileRate(1)
		defer flushBlockProfile(ctx, blockProfilePath)
		hooks = append(hooks, func(os.Signal) { flushBlockProfile(ctx, blockProfilePath) })
	}

	if stackTrace {
		libruntime.InstallStackTraceOnInterrupt()
	}
	for _, hook := range hooks {
		libruntime.InstallInterruptHook(hook)
	}

	// start telemetry
	shutdown, err := xruntime.SetupTelemetry(ctx, instanceID, metricsAddr, pprofEnabled)
	if err != nil {
		liblog.Fatalf(ctx, "starting telemetry: %v", err)
		return err
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			liblog.Fatalf(ctx, "shutdown telemetry: %v", err)
		}
	}()
	hooks = append(hooks,
		func(os.Signal) {
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

var pprofEnabled bool
var pprofMutexFraction int
var pprofBlockRate int

func SetPprof(enabled bool, mutexFraction, blockRate int) {
	pprofEnabled = enabled
	pprofMutexFraction = mutexFraction
	pprofBlockRate = blockRate
}

func applyPprofRates(setMutex func(int) int, setBlock func(int)) {
	if pprofEnabled {
		setMutex(pprofMutexFraction)
		setBlock(pprofBlockRate)
	}
}

var stackTrace bool

func SetStackTrace(s bool) {
	stackTrace = s
}
