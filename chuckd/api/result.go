package api

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
)

type Result struct {
	Returned any    `json:"returned,omitempty"`
	Error    error  `json:"error,omitempty"` // structure of error
	Msg      string `json:"msg,omitempty"`   // summary of error
}

func NewResult(r any, err *must.Error) Result {
	var result Result
	if err != nil {
		result.Msg = err.Error()
		result.Error = err.Unwrap()
	}
	result.Returned = r
	return result
}

func (r Result) OsExitCode() int {
	if r.Error != nil {
		return 1
	}
	return 0
}

func Invoke(f func()) Result {

	// exit on error, but after dumping cpu and mem profiles
	var xerr *must.Error
	defer func() {
		if xerr != nil {
			os.Exit(1)
		}
	}()

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

	//
	xerr = must.TryThru(f)
	r := NewResult(nil, xerr)
	if xerr != nil {
		if log.IsVerbose() {
			fmt.Fprintln(os.Stderr, xerr.StackString())
		}
		fmt.Fprintln(os.Stderr, xerr.Error())
	}

	return r
}

func Invoke1[R1 any](f func() R1) Result {

	// exit on error, but after dumping cpu and mem profiles
	var xerr *must.Error
	defer func() {
		if xerr != nil {
			os.Exit(1)
		}
	}()

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

	//
	var r1 R1
	r1, xerr = must.Try1Thru[R1](f)
	r := NewResult(r1, xerr)
	if xerr != nil {
		if log.IsVerbose() {
			fmt.Fprintln(os.Stderr, xerr.StackString())
		}
		fmt.Fprintln(os.Stderr, xerr.Error())
	}
	fmt.Fprint(os.Stdout, form.SprintJSON(r.Returned))

	return r
}

var cpuProfilePath string

func SetCPUProfilePath(filepath string) {
	cpuProfilePath = filepath
}

var memProfilePath string

func SetMemProfilePath(filepath string) {
	memProfilePath = filepath
}
