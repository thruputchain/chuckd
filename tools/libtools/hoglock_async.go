package libtools

import (
	"context"
	"fmt"
	"sort"
)

type AsyncLock struct{}

var AsyncHogLock AsyncLock

func (x AsyncLock) Lock(ctx context.Context, holder string, host string) error {

	scriptFmt := `
set -euo pipefail

lock_dir=/run/user/$(id -u $(whoami))/lock
lock_file="$lock_dir/holder"

while ! mkdir -p "$lock_dir" 2>/dev/null; do
  sleep 0.1
done

echo %q > "$lock_file"

echo "lock acquired: $lock_dir"
exit 0
`
	script := fmt.Sprintf(scriptFmt, holder)
	_, err := SshExecBlock(ctx, host, script)
	return err
}

func (x AsyncLock) LockMany(ctx context.Context, holder string, hosts []string) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sort.Strings(hosts)

	ech := make(chan error, len(hosts))
	for _, host := range hosts {
		go func(host string) {
			ech <- x.Lock(ctx, holder, host)
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

func (AsyncLock) Unlock(ctx context.Context, holder string, host string) error {

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
	_, err := SshExecBlock(ctx, host, script)
	return err
}

func (x AsyncLock) UnlockMany(ctx context.Context, holder string, hosts []string) error {

	sort.Strings(hosts)

	ech := make(chan error, len(hosts))
	for _, host := range hosts {
		go func(host string) {
			ech <- x.Unlock(ctx, holder, host)
		}(host)
	}

	var someErr error
	for i := 0; i < len(hosts); i++ {
		if err := <-ech; err != nil {
			someErr = err
		}
	}

	return someErr
}
