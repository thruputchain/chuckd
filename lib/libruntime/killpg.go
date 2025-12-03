package libruntime

import (
	"os"
	"syscall"

	"github.com/groundhogchain/chuckd/lib/liblog"
)

func InstallKillProcessGroupOnInterrupt() {
	liblog.Infof("installing kill process group on SIGINT and SIGTERM")
	syscall.Setpgid(0, 0)
	RegisterInterruptHook(
		func(sig os.Signal) {
			pgid, _ := syscall.Getpgid(0)
			// negative PID means "process group"
			syscall.Kill(-pgid, sig.(syscall.Signal))
			os.Exit(177)
		},
	)
}
