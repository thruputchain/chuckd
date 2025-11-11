package libtools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/textutil"
)

func Exec(
	ctx context.Context,
	script string,
) (*bytes.Buffer, *bytes.Buffer, func() error) {
	return ExecReader(ctx, bytes.NewBufferString(script))
}

// XXX: if the ctx is cancelled, kill the process
func ExecReader(
	ctx context.Context,
	script io.Reader,
) (*bytes.Buffer, *bytes.Buffer, func() error) {

	bashPath, err := exec.LookPath("bash")
	must.NoError(ctx, err)

	bashOpts := []string{"-s"}

	cmd := exec.Command(bashPath, bashOpts...)
	cmd.Env = []string{}
	cmd.Stdin = io.MultiReader(
		bytes.NewBufferString("trap 'kill 0' EXIT\n(\n"),
		script,
		bytes.NewBufferString("\n)\n"),
	)

	outbuf, errbuf := &bytes.Buffer{}, &bytes.Buffer{}
	stdout := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, outbuf), "[out] ")
	stderr := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, errbuf), "[err] ")

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Start()
	must.NoError(ctx, err)

	waitCh := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			err := cmd.Process.Kill()
			must.NoError(ctx, err)
		case <-waitCh:
		}
	}()

	return outbuf, errbuf, func() error {
		err := cmd.Wait()
		waitCh <- err
		return cmd.Wait()
	}
}

func ExecBlock(
	ctx context.Context,
	script string,
) (string, string, error) {
	outb, errb, wait := Exec(ctx, script)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec_block error (%v):\n", err)
	}
	return outb.String(), errb.String(), err
}

func ExecReaderBlock(
	ctx context.Context,
	script io.Reader,
) (string, string, error) {
	outb, errb, wait := ExecReader(ctx, script)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec_block error (%v):\n", err)
	}
	return outb.String(), errb.String(), err
}
