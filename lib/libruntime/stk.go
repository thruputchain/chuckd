package libruntime

import (
	"os"
	"os/signal"
	"runtime"
	"slices"
	"sync"
	"syscall"

	"github.com/groundhogchain/chuckd/lib/liblog"
)

var (
	signalLk    sync.Mutex
	signalHooks []func(sig os.Signal)
)

func RegisterInterruptHook(f func(sig os.Signal)) {
	signalLk.Lock()
	defer signalLk.Unlock()
	signalHooks = append(signalHooks, f)
}

func init() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		println("signal: ", sig.String())

		signalLk.Lock()
		hooks := slices.Clone(signalHooks)
		signalLk.Unlock()

		for _, hook := range hooks {
			println("calling hook")
			hook(sig)
		}
		os.Exit(1)
	}()
}

func InstallStackTraceOnInterrupt() {
	liblog.Infof("installing stack trace on SIGINT and SIGTERM")
	RegisterInterruptHook(
		func(sig os.Signal) {
			buf := make([]byte, 1<<20)
			stackLen := runtime.Stack(buf, true)
			os.Stderr.Write(buf[:stackLen])
		},
	)
}
