package runtime

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func CaptureInterrupts(hook ...func()) {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		println("signal:", sig)

		for _, hook := range hook {
			hook()
		}

		buf := make([]byte, 1<<20)
		stackLen := runtime.Stack(buf, true)
		os.Stderr.Write(buf[:stackLen])
		os.Exit(1)
	}()
}
