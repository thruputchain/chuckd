package libtools

import (
	"context"
	"fmt"
	"os"
	"sort"
)

func HogLock(ctx context.Context, holder string, host string) error {

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
	stdout, stderr, err := SshExecBlock(ctx, host, script)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hog_lock holder=%s host=%s stdout:\n%s\n", holder, host, stdout)
		fmt.Fprintf(os.Stderr, "hog_lock holder=%s host=%s stderr:\n%s\n", holder, host, stderr)
	}
	return err
}

func HogLockMany(ctx context.Context, holder string, hosts []string) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sort.Strings(hosts)

	ech := make(chan error, len(hosts))
	for _, host := range hosts {
		go func(host string) {
			ech <- HogLock(ctx, holder, host)
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
