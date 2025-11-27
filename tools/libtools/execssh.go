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

	sshOpts := []string{`-tt`, `-o`, `BatchMode=yes`, `-o`, `ConnectTimeout=5`, `-o`, `VisualHostKey=no`, `-A`}
	sshOpts = append(sshOpts, host, `setsid`, `-w`, `bash`, `-s`)

	cmd := exec.CommandContext(ctx, sshPath, sshOpts...)
	cmd.Env = nil // nil ensures that SSH_AUTH_SOCK is passed to ssh

	var prefix bytes.Buffer
	// Disable TTY echo and prompt (we use -tt for SIGHUP on disconnect, but don't want the noise)
	fmt.Fprintln(&prefix, `stty -echo`)
	fmt.Fprintln(&prefix, `PS1=`)
	fmt.Fprintln(&prefix, `set -euo pipefail`)

	fmt.Fprintln(&prefix, `trap 'echo "GOT EXIT\n" >> /tmp/sigs.log; trap "" TERM HUP INT; kill 0 2>/dev/null || true' EXIT`)
	fmt.Fprintln(&prefix, `trap 'echo "GOT SIGINT\n" >> /tmp/sigs.log; exit 130' INT`)   // 128 + 2 (SIGINT)
	fmt.Fprintln(&prefix, `trap 'echo "GOT SIGTERM\n" >> /tmp/sigs.log; exit 143' TERM`)  // 128 + 15 (SIGTERM)
	fmt.Fprintln(&prefix, `trap 'echo "GOT SIGHUP\n" >> /tmp/sigs.log; exit 129' HUP`)   // 128 + 1 (SIGHUP)
	for name, value := range env {
		fmt.Fprintf(&prefix, "export %s=%q\n", name, value)
	}

	fullScript, err := io.ReadAll(io.MultiReader(&prefix, script, bytes.NewBufferString("\nexit $?\n")))
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
	if exitErr, ok := err.(*exec.ExitError); ok {
		fmt.Fprintf(os.Stderr, "[%s] exit code %d\n", host, exitErr.ExitCode())
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] ssh error: %v\n", host, err)
	} else {
		fmt.Fprintf(os.Stderr, "[%s] exit code 0\n", host)
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
	if exitErr, ok := err.(*exec.ExitError); ok {
		fmt.Fprintf(os.Stderr, "[%s] exit code %d\n", host, exitErr.ExitCode())
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] ssh error: %v\n", host, err)
	} else {
		fmt.Fprintf(os.Stderr, "[%s] exit code 0\n", host)
	}
	return outb.String(), errb.String(), err
}
