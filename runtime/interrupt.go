package runtime

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func CaptureInterrupts() {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		println("signal:", sig)

		buf := make([]byte, 1<<20)
		stackLen := runtime.Stack(buf, true)
		os.Stderr.Write(buf[:stackLen])
		os.Exit(1)
	}()
}
