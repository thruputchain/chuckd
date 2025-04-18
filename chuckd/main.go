package main

import (
	"os"

	"github.com/groundhogchain/chuckd/chuckd/cmd"
)

func main() {
	if err := cmd.Execute(); err == nil {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
