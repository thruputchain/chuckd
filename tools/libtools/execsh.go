package libtools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/groundhogchain/chuckd/util/textutil"
	"github.com/groundhogchain/lib-go/must"
)

// if ctx is cancelled, the process will be killed
func ExecReader(
	ctx context.Context,
	script io.Reader,
	env map[string]string,
) (*bytes.Buffer, *bytes.Buffer, func() error) {

	bashPath, err := exec.LookPath("bash")
	must.NoError(ctx, err)

	bashOpts := []string{"-s"}

	cmd := exec.CommandContext(ctx, bashPath, bashOpts...)
	cmd.Env = nil // nil ensures that bash inherits envs

	var prefix bytes.Buffer
	fmt.Fprintln(&prefix, `set -euo pipefail`)
	fmt.Fprintln(&prefix, `trap 'status=$?; if [ $status -ne 0 ]; then kill 0; fi' EXIT`)
	for name, value := range env {
		fmt.Fprintf(&prefix, "export %s=%q\n", name, value)
	}
	fmt.Fprintln(&prefix, `(`)

	cmd.Stdin = io.MultiReader(&prefix, script, bytes.NewBufferString("\n)\n"))

	outbuf, errbuf := &bytes.Buffer{}, &bytes.Buffer{}
	stdout := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, outbuf), "[out] ")
	stderr := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, errbuf), "[err] ")

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = cmd.Start()
	must.NoError(ctx, err)

	waitCh := make(chan error, 1)
	ck.register(cmd.Process.Pid)
	go func() {
		select {
		case <-ctx.Done():
		case <-waitCh:
		}
		fmt.Fprintf(os.Stderr, "shell_exec killing pid %d\n", cmd.Process.Pid)
		syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		ck.unregister(cmd.Process.Pid)
	}()

	return outbuf, errbuf, func() error {
		err := cmd.Wait()
		waitCh <- err
		return err
	}
}

func Exec(
	ctx context.Context,
	script string,
	env map[string]string,
) (*bytes.Buffer, *bytes.Buffer, func() error) {
	return ExecReader(ctx, bytes.NewBufferString(script), env)
}

func ExecBlock(
	ctx context.Context,
	script string,
	env map[string]string,
) (string, string, error) {
	outb, errb, wait := Exec(ctx, script, env)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec_block error (%v):\n", err)
	}
	return outb.String(), errb.String(), err
}

func ExecReaderBlock(
	ctx context.Context,
	script io.Reader,
	env map[string]string,
) (string, string, error) {
	outb, errb, wait := ExecReader(ctx, script, env)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "exec_block error (%v):\n", err)
	}
	return outb.String(), errb.String(), err
}
