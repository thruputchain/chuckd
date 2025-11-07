package libtools

import (
	"context"
	"fmt"
	"os"
	"sort"
)

func HogUnlock(ctx context.Context, holder string, host string) error {

	scriptFmt := `
set -euo pipefail

lock_dir=/run/user/$(id -u "$(whoami)")/lock
lock_file="$lock_dir/holder"

if [ ! -f "$lock_file" ]; then
  echo "no lock file at $lock_file" >&2
  exit 1
fi

# Read and trim the contents of the holder file
owner=$(cat "$lock_file" | tr -d '[:space:]')

if [ "$owner" != %q ]; then
  echo "lock held by '$owner', not '%s'" >&2
  exit 2
fi

# Contents match: remove the file and directory
rm -f -- "$lock_file"
rmdir --ignore-fail-on-non-empty -- "$lock_dir" || true

echo "lock released: $lock_dir"
exit 0
`

	script := fmt.Sprintf(scriptFmt, holder, holder)
	stdout, stderr, err := SshExecBlock(ctx, host, script)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hog_unlock holder=%s host=%s stdout:\n%s\n", holder, host, stdout)
		fmt.Fprintf(os.Stderr, "hog_unlock holder=%s host=%s stderr:\n%s\n", holder, host, stderr)
	}
	return err
}

func HogUnlockMany(ctx context.Context, holder string, hosts []string) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sort.Strings(hosts)

	ech := make(chan error, len(hosts))
	for _, host := range hosts {
		go func(host string) {
			ech <- HogUnlock(ctx, holder, host)
		}(host)
	}

	for i := 0; i < len(hosts); i++ {
		if err := <-ech; err != nil {
			cancel()
			return err
		}
	}

	return nil
}
