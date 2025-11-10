package libtools

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
) (*bytes.Buffer, *bytes.Buffer, func() error) {
	return SshExecReader(ctx, host, bytes.NewBufferString(script))
}

func SshExecReader(
	ctx context.Context,
	host string,
	script io.Reader,
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
	cmd.Stdin = script

	outbuf, errbuf := &bytes.Buffer{}, &bytes.Buffer{}
	stdout := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, outbuf), "[out] ")
	stderr := textutil.NewPrefixWriter(io.MultiWriter(os.Stdout, errbuf), "[err] ")

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Start()
	must.NoError(ctx, err)

	return outbuf, errbuf, func() error { return cmd.Wait() }
}

func SshExecBlock(
	ctx context.Context,
	host string,
	script string,
) (string, string, error) {
	outb, errb, wait := SshExec(ctx, host, script)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh_exec_block error (%v):\n", err)
	}
	return outb.String(), errb.String(), err
}

func SshExecReaderBlock(
	ctx context.Context,
	host string,
	script io.Reader,
) (string, string, error) {
	outb, errb, wait := SshExecReader(ctx, host, script)
	err := wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh_exec_block error (%v):\n", err)
	}
	return outb.String(), errb.String(), err
}
