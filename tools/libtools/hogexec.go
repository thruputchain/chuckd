package libtools

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

func HogExecReader(
	ctx context.Context,
	jid string,
	host string,
	script io.Reader,
) (string, string, error) {

	jdir := JobDir(jid)
	prefix := fmt.Sprintf(`
set -euo pipefail
mkdir -p %q
cd %q
`, jdir, jdir)
	r := io.MultiReader(bytes.NewBufferString(prefix), script)
	return SshExecReaderBlock(ctx, host, r)
}

func HogExec(
	ctx context.Context,
	jid string,
	host string,
	script string,
) (string, string, error) {

	jdir := JobDir(jid)
	prefix := fmt.Sprintf(`
set -euo pipefail
mkdir -p %q
cd %q
`, jdir, jdir)
	return SshExecBlock(ctx, host, prefix+script)
}
