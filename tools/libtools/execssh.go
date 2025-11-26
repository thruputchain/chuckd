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

	// Use -tt to force TTY allocation - this ensures bash receives SIGHUP when SSH disconnects
	sshOpts := []string{`-tt`, `-o`, `BatchMode=yes`, `-o`, `ConnectTimeout=5`, `-o`, `VisualHostKey=no`, `-A`}
	sshOpts = append(sshOpts, host, `bash`, `-s`)

	cmd := exec.CommandContext(ctx, sshPath, sshOpts...)
	cmd.Env = nil // nil ensures that SSH_AUTH_SOCK is passed to ssh

	var prefix bytes.Buffer
	fmt.Fprintln(&prefix, `set -euo pipefail`)
	// Trap signals to kill all processes in the shell's process group
	fmt.Fprintln(&prefix, `trap 'status=$?; kill 0 2>/dev/null || true; echo "hit trap handle 1";exit $status' INT TERM HUP`)
	for name, value := range env {
		fmt.Fprintf(&prefix, "export %s=%q\n", name, value)
	}
	// Run script in subshell - set trap here too so it executes when SSH disconnects
	fmt.Fprintln(&prefix, `(`)
	fmt.Fprintln(&prefix, `trap 'status=$?; kill 0 2>/dev/null || true; echo "hit trap handle 2"; exit $status' INT TERM HUP`)

	fullScript, err := io.ReadAll(io.MultiReader(&prefix, script, bytes.NewBufferString("\n)\n")))
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
		if err == nil {
			liblog.Infof("ssh_exec on %s completed successfully", host)
		} else {
			liblog.Errorf("ssh_exec on %s failed (%v)", host, err)
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
