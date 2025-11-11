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

func SshExec(
	ctx context.Context,
	host string,
	script string,
) (*bytes.Buffer, *bytes.Buffer, func() error) {
	return SshExecReader(ctx, host, bytes.NewBufferString(script))
}

// if ctx is cancelled, the process will be killed
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
		return err
	}
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
