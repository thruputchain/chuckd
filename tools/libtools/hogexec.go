package libtools

import (
	"context"
	"fmt"
)

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
