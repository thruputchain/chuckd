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
	fmt.Fprintln(&prefix, `trap 'kill 0 2>/dev/null || true; echo "trapped"; exit' INT TERM HUP`)
	for name, value := range env {
		fmt.Fprintf(&prefix, "export %s=%q\n", name, value)
	}
	// Run user script in subshell in background, then wait for it while also monitoring stdin
	// When stdin closes (SSH dies), the read fails and we kill the background job
	fmt.Fprintln(&prefix, `(`)

	var suffix bytes.Buffer
	fmt.Fprintln(&suffix, ``)
	fmt.Fprintln(&suffix, `) &`)
	fmt.Fprintln(&suffix, `SCRIPT_PID=$!`)
	fmt.Fprintln(&suffix, `# Save stdin to fd 3 before backgrounding (background jobs get stdin from /dev/null)`)
	fmt.Fprintln(&suffix, `exec 3<&0`)
	fmt.Fprintln(&suffix, `# Monitor fd 3 (original stdin) - read blocks until pipe closes (SSH dies), then kill everything`)
	fmt.Fprintln(&suffix, `{ read -r _ <&3 2>/dev/null; echo "read failed"; kill 0 2>/dev/null; } &`)
	fmt.Fprintln(&suffix, `wait $SCRIPT_PID 2>/dev/null`)
	fmt.Fprintln(&suffix, `STATUS=$?`)
	fmt.Fprintln(&suffix, `echo "status: $STATUS"`)
	fmt.Fprintln(&suffix, `exit $STATUS`)

	fullScript, err := io.ReadAll(io.MultiReader(&prefix, script, &suffix))
	must.NoError(ctx, err)
	liblog.Debugf("ssh_exec executing:\n%s\n", fullScript)

	// Use a pipe for stdin so we can keep it open
	stdinPipe, err := cmd.StdinPipe()
	must.NoError(ctx, err)

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

	// Write the script to stdin pipe
	_, err = stdinPipe.Write(fullScript)
	must.NoError(ctx, err)
	// Don't close stdinPipe - keep it open so the remote can detect when we die

	pid := cmd.Process.Pid
	// Register negative PID to kill the process group
	ck.register(-pid)

	// Kill SSH process group only when context is cancelled
	go func() {
		<-ctx.Done()
		fmt.Fprintf(os.Stderr, "ssh_exec killing process group %d due to context cancellation\n", pid)
		// Close stdin pipe first - this triggers the remote cleanup
		stdinPipe.Close()
		// Then kill the entire process group
		syscall.Kill(-pid, syscall.SIGKILL)
	}()

	return outbuf, errbuf, func() error {
		err := cmd.Wait()
		ck.unregister(-pid)
		stdinPipe.Close() // Close on normal exit too
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
