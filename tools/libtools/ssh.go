package libtools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/textutil"
)

func SshExec(
	ctx context.Context,
	host string,
	script string,
) (*bytes.Buffer, func() error) {

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
	out := &bytes.Buffer{}
	cmd.Stdout = textutil.NewPrefixWriter(out, "[out] ")
	cmd.Stderr = textutil.NewPrefixWriter(out, "[err] ")

	err = cmd.Start()
	must.NoError(ctx, err)

	return out, func() error { return cmd.Wait() }
}

func SshExecBlock(
	ctx context.Context,
	host string,
	script string,
) (string, error) {
	out, wait := SshExec(ctx, host, script)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh_exec_block error (%v):\n", err)
		fmt.Fprintln(os.Stderr, out)
	}
	return out.String(), err
}
