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

	// Use -tt for PTY so bash receives SIGHUP when SSH disconnects
	sshOpts := []string{`-tt`, `-o`, `BatchMode=yes`, `-o`, `ConnectTimeout=5`, `-o`, `VisualHostKey=no`, `-A`}
	sshOpts = append(sshOpts, host, `bash`, `-s`)

	cmd := exec.CommandContext(ctx, sshPath, sshOpts...)
	cmd.Env = nil // nil ensures that SSH_AUTH_SOCK is passed to ssh

	var prefix bytes.Buffer
	// MUST be first line - disable TTY echo before any other output
	fmt.Fprintln(&prefix, `stty -echo 2>/dev/null || true`)
	fmt.Fprintln(&prefix, `set -euo pipefail`)
	fmt.Fprintln(&prefix, `trap 'kill 0 2>/dev/null || true; exit' INT TERM HUP`)
	for name, value := range env {
		fmt.Fprintf(&prefix, "export %s=%q\n", name, value)
	}
	fmt.Fprintln(&prefix, `(`)

	fullScript, err := io.ReadAll(io.MultiReader(&prefix, script, bytes.NewBufferString("\n)\n")))
	must.NoError(ctx, err)
	liblog.Debugf("ssh_exec executing:\n%s\n", fullScript)

	cmd.Stdin = bytes.NewBuffer(fullScript)

	outbuf, errbuf := &bytes.Buffer{}, &bytes.Buffer{}
	out_prefix := fmt.Sprintf("[%s:out] ", host)
	err_prefix := fmt.Sprintf("[%s:err] ", host)
	stdout := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, outbuf), out_prefix)
	stderr := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, errbuf), err_prefix)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Put SSH in its own process group so we can kill it and all its children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = cmd.Start()
	must.NoError(ctx, err)

	pid := cmd.Process.Pid
	// Register negative PID to kill the process group
	ck.register(-pid)

	// Kill SSH process group only when context is cancelled
	go func() {
		<-ctx.Done()
		fmt.Fprintf(os.Stderr, "ssh_exec killing process group %d due to context cancellation\n", pid)
		// Kill the entire process group - SSH dies, PTY closes, bash gets SIGHUP
		syscall.Kill(-pid, syscall.SIGKILL)
	}()

	return outbuf, errbuf, func() error {
		err := cmd.Wait()
		ck.unregister(-pid)
		if err == nil {
			fmt.Fprintf(os.Stderr, "ssh_exec on %s completed successfully\n", host)
		}
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
