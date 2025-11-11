package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/tools/libtools"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "hog_par",
		Usage: "execute scripts in parallel",
		Flags: []cli.Flag{},
		Action: func(ctx context.Context, c *cli.Command) error {

			lines, err := ReadLines(os.Stdin)
			must.NoError(ctx, err)

			return libtools.HogPar(ctx, lines)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func ReadLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
