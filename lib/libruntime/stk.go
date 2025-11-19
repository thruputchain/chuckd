package libruntime

import (
	"os"
	"os/signal"
	"runtime"
	"slices"
	"sync"
	"syscall"
)

var (
	signalLk    sync.Mutex
	signalHooks []func()
)

func RegisterInterruptHook(f func()) {
	signalLk.Lock()
	defer signalLk.Unlock()
	signalHooks = append(signalHooks, f)
}

func init() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan

		signalLk.Lock()
		hooks := slices.Clone(signalHooks)
		signalLk.Unlock()

		for _, hook := range hooks {
			hook()
		}
		os.Exit(1)
	}()
}

func StackTraceInterrupt() {
	RegisterInterruptHook(func() {
		buf := make([]byte, 1<<20)
		stackLen := runtime.Stack(buf, true)
		os.Stderr.Write(buf[:stackLen])
	})
}
