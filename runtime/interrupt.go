package runtime

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func CaptureInterrupts(stackTrace bool, hook ...func()) {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		println("signal:", sig)

		for _, hook := range hook {
			hook()
		}

		if stackTrace {
			buf := make([]byte, 1<<20)
			stackLen := runtime.Stack(buf, true)
			os.Stderr.Write(buf[:stackLen])
		}
		os.Exit(1)
	}()
}
