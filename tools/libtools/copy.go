package libtools

import (
	"bytes"
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/must"
)

func Copy(
	ctx context.Context,
	from string, // from host
	to string, // to host
	in []string, // source paths (files or dirs)
	out []string, // destination paths
) error {

	must.Assert(ctx, len(in) == len(out))
	if len(in) == 0 {
		return nil
	}

	var w bytes.Buffer
	for i := range out {
		fmt.Fprintf(&w, "rsync --mkpath -r %q %s:%q\n", in[i], to, out[i])
	}

	_, _, err := SshExecBlock(ctx, from, w.String())
	return err
}

func CopyFrom(
	ctx context.Context,
	from string, // from host
	in []string, // source paths (files or dirs)
	out []string, // destination paths
) error {

	must.Assert(ctx, len(in) == len(out))
	if len(in) == 0 {
		return nil
	}

	var w bytes.Buffer
	for i := range out {
		fmt.Fprintf(&w, "rsync --mkpath -r %s:%q %q\n", from, in[i], out[i])
	}

	_, _, err := ExecBlock(ctx, w.String())
	return err
}

func CopyTo(
	ctx context.Context,
	to string, // from host
	in []string, // source paths (files or dirs)
	out []string, // destination paths
) error {

	must.Assert(ctx, len(in) == len(out))
	if len(in) == 0 {
		return nil
	}

	var w bytes.Buffer
	for i := range out {
		fmt.Fprintf(&w, "rsync --mkpath -r %q %s:%q\n", in[i], to, out[i])
	}

	_, _, err := ExecBlock(ctx, w.String())
	return err
}
