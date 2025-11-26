package libtools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/textutil"
)

// if ctx is cancelled, the process will be killed
func SshExecReader(
	ctx context.Context,
	host string,
	script io.Reader,
	env map[string]string,
) (*bytes.Buffer, *bytes.Buffer, func() error) {

	sshPath, err := exec.LookPath("ssh")
	must.NoError(ctx, err)

	sshOpts := []string{`-T`, `-o`, `BatchMode=yes`, `-o`, `ConnectTimeout=5`, `-o`, `VisualHostKey=no`, `-A`}
	sshOpts = append(sshOpts, host, `bash`, `-s`)

	cmd := exec.CommandContext(ctx, sshPath, sshOpts...)
	cmd.Env = nil // nil ensures that SSH_AUTH_SOCK is passed to ssh

	var prefix bytes.Buffer
	fmt.Fprintln(&prefix, `set -euo pipefail`)
	// Kill entire process group when bash exits. This handles both normal exit
	// and when SSH connection closes (bash sees EOF). Yes, this kills bash too,
	// but that's fine - the important thing is children die.
	fmt.Fprintln(&prefix, `trap 'kill 0 2>/dev/null' EXIT`)
	for name, value := range env {
		fmt.Fprintf(&prefix, "export %s=%q\n", name, value)
	}

	fullScript, err := io.ReadAll(io.MultiReader(&prefix, script, bytes.NewBufferString("\n")))
	must.NoError(ctx, err)
	stdin := bytes.NewBufferString(string(fullScript))
	liblog.Debugf("ssh_exec executing:\n%s\n", fullScript)
	// stdin := io.MultiReader(&prefix, script, bytes.NewBufferString("\n)\n"))

	cmd.Stdin = stdin

	outbuf, errbuf := &bytes.Buffer{}, &bytes.Buffer{}
	out_prefix := fmt.Sprintf("[%s:out] ", host)
	err_prefix := fmt.Sprintf("[%s:err] ", host)
	stdout := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, outbuf), out_prefix)
	stderr := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, errbuf), err_prefix)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = cmd.Start()
	must.NoError(ctx, err)

	waitCh := make(chan error, 1)
	ck.register(cmd.Process.Pid)
	go func() {
		select {
		case <-ctx.Done():
		case <-waitCh:
		}
		fmt.Fprintf(os.Stderr, "ssh_exec killing pid %d\n", cmd.Process.Pid)
		syscall.Kill(cmd.Process.Pid, syscall.SIGTERM)
		ck.unregister(cmd.Process.Pid)
	}()

	return outbuf, errbuf, func() error {
		err := cmd.Wait()
		waitCh <- err
		return err
	}
}

func SshExec(
	ctx context.Context,
	host string,
	script string,
	env map[string]string,
) (*bytes.Buffer, *bytes.Buffer, func() error) {
	return SshExecReader(ctx, host, bytes.NewBufferString(script), env)
}

func SshExecBlock(
	ctx context.Context,
	host string,
	script string,
	env map[string]string,
) (string, string, error) {
	outb, errb, wait := SshExec(ctx, host, script, env)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh_exec error (%v):\n", err)
	}
	return outb.String(), errb.String(), err
}

func SshExecReaderBlock(
	ctx context.Context,
	host string,
	script io.Reader,
	env map[string]string,
) (string, string, error) {
	outb, errb, wait := SshExecReader(ctx, host, script, env)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh_exec error (%v):\n", err)
	}
	return outb.String(), errb.String(), err
}
