package main

import (
	"os"

	"github.com/groundhogchain/chuckd/chuckd/api"
	"github.com/groundhogchain/chuckd/chuckd/cmd"
	_ "github.com/groundhogchain/chuckd/runtime"
)

func main() {
	r := api.Invoke(func() { cmd.Execute() })
	os.Exit(r.OsExitCode())
}
