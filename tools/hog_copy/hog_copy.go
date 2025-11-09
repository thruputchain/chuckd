package main

import (
	"fmt"
	"os"
	"strings"
)

type FilePair struct {
	From string
	To   string
}

func main() {
	id := ""
	var fromArgs, toArgs []string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--id="):
			id = strings.TrimPrefix(arg, "--id=")

		case strings.HasPrefix(arg, "--from="):
			fromArgs = append(fromArgs, strings.TrimPrefix(arg, "--from="))

		case strings.HasPrefix(arg, "--to="):
			toArgs = append(toArgs, strings.TrimPrefix(arg, "--to="))

		default:
			fmt.Fprintf(os.Stderr, "unknown argument: %s\n", arg)
			os.Exit(1)
		}
	}

	if len(fromArgs) != len(toArgs) {
		fmt.Fprintf(os.Stderr, "mismatched number of --from and --to arguments\n")
		os.Exit(1)
	}

	pairs := make([]FilePair, len(fromArgs))
	for i := range fromArgs {
		pairs[i] = FilePair{
			From: fromArgs[i],
			To:   toArgs[i],
		}
	}

	fmt.Printf("ID: %s\n", id)
	for i, p := range pairs {
		fmt.Printf("Pair %d: from %s → to %s\n", i+1, p.From, p.To)
	}
}
