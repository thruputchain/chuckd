package main

import (
	"os"
	"syscall"

	"github.com/groundhogchain/chuckd/chuckd/cmd"
)

func main() {
	syscall.Setpgid(0, 0)
	if err := cmd.Execute(); err == nil {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
