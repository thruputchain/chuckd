package libtools

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"github.com/groundhogchain/chuckd/lib/must"
)

func HogCopy(
	ctx context.Context,
	jid string, // job id
	from string, // from host
	to string, // to host
	in []string, // source paths (files or dirs)
	out []string, // destination paths
) error {

	must.Assert(ctx, len(in) == len(out))
	if len(in) == 0 {
		return nil
	}

	jdir := JobDir(jid)

	var w bytes.Buffer
	for i := range out {
		fmt.Fprintf(&w, "set -ex\n")
		fmt.Fprintf(&w, "rsync --mkpath -r %q %s:%q\n",
			path.Join(jdir, in[i]),
			to,
			path.Join(jdir, out[i]),
		)
	}

	_, _, err := SshExecBlock(ctx, from, w.String())
	return err
}

func HogCopyFrom(
	ctx context.Context,
	jid string, // job id
	from string, // from host
	in []string, // source paths (files or dirs)
	out []string, // destination paths
) error {

	must.Assert(ctx, len(in) == len(out))
	if len(in) == 0 {
		return nil
	}

	jdir := JobDir(jid)

	var w bytes.Buffer
	for i := range out {
		fmt.Fprintf(&w, "set -ex\n")
		fmt.Fprintf(&w, "rsync --mkpath -r %s:%q %q\n",
			from,
			path.Join(jdir, in[i]),
			path.Join(".", out[i]),
		)
	}

	_, _, err := ExecBlock(ctx, w.String())
	return err
}

func HogCopyTo(
	ctx context.Context,
	jid string, // job id
	to string, // from host
	in []string, // source paths (files or dirs)
	out []string, // destination paths
) error {

	must.Assert(ctx, len(in) == len(out))
	if len(in) == 0 {
		return nil
	}

	jdir := JobDir(jid)

	var w bytes.Buffer
	for i := range out {
		fmt.Fprintf(&w, "set -ex\n")
		fmt.Fprintf(&w, "rsync --mkpath -r %q %s:%q\n",
			path.Join(".", in[i]),
			to,
			path.Join(jdir, out[i]),
		)
	}

	_, _, err := ExecBlock(ctx, w.String())
	return err
}
