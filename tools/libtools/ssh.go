package libtools

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
)

func SshExec(
	ctx context.Context,
	host string,
	script string,
) (*bytes.Buffer, *bytes.Buffer, func() error) {

	sshPath, err := exec.LookPath("ssh")
	must.NoError(ctx, err)

	sshOpts := []string{`-T`, `-o`, `BatchMode=yes`, `-o`, `ConnectTimeout=5`, `-o`, `VisualHostKey=no`}
	sshOpts = append(sshOpts, host, `bash -s`)

	cmd := exec.Command(sshPath, sshOpts...)
	cmd.Env = []string{
		"RUST_BACKTRACE=1",
		"RUST_LOG=" + runtime.GetLogLevel(),
	}
	cmd.Stdin = bytes.NewBufferString(script)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr

	err = cmd.Start()
	must.NoError(ctx, err)

	return stdout, stderr, func() error { return cmd.Wait() }
}

func SshExecBlock(
	ctx context.Context,
	host string,
	script string,
) (string, string, error) {
	stdout, stderr, wait := SshExec(ctx, host, script)
	return stdout.String(), stderr.String(), wait()
}
