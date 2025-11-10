package libtools

import (
	"context"
	"sort"

	"github.com/groundhogchain/chuckd/util/textutil"
)

type SyncLock struct{}

var SyncHogLock SyncLock

func (x SyncLock) HoldLock(ctx context.Context, holder string, host string) error {

	scriptTmpl := `
set -euo pipefail

lock_dir=/run/user/$(id -u $(whoami))/lock
lock_file="$lock_dir/lock"
holder_file="$lock_dir/holder"

mkdir -p -- "$dir"

# Open the lock file on fd 9 and take an exclusive lock.
# We don't store data in the lock file; it's just for locking.
exec 9>"$lock_file"
flock -x 9

stage_file="$(mktemp "${lock_dir}/holder.staging.XXXXXX")"

cleanup() {
  rm -f -- "$stage_file" 2>/dev/null || true
  # Close fd 9 (this also releases the lock if we're still running)
  exec 9>&- 2>/dev/null || true
}
trap cleanup EXIT

# Write the new content into the temp file
printf '%s\n' {{ printf "%q" .Holder }} >"$stage_file"

# (Optional) best-effort sync
if command -v fsync >/dev/null 2>&1; then
  fsync "$stage_file" || true
fi

mv -f -- "$stage_file" "$data_file"

echo "acquired lock on $lock_file for {{ .Holder }}, as pid: $$"
echo "hanging forever; kill this process to release the lock."

while :; do
  sleep 86400
done
`
	script := textutil.Apply(ctx, scriptTmpl, textutil.M{
		"Holder": holder,
	})
	_, _, err := SshExecBlock(ctx, host, script)
	return err
}

func (x SyncLock) HoldLockMany(ctx context.Context, holder string, hosts []string) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sort.Strings(hosts)

	ech := make(chan error, len(hosts))
	for _, host := range hosts {
		go func(host string) {
			ech <- x.HoldLock(ctx, holder, host)
		}(host)
	}

	err := <-ech
	cancel()
	return err
}

func (x SyncLock) WaitLock(ctx context.Context, holder string, host string) error {

	scriptTmpl := `
set -euo pipefail

expected={{ printf "%q" .Holder }}

lock_dir=/run/user/$(id -u $(whoami))/lock
holder_file="$lock_dir/holder"

while true; do
  if [[ -f "$holder_file" ]]; then
    content=$(<"$holder_file")
    if [[ "$content" == "$expected" ]]; then
      echo "file '$holder_file' now contains: '$expected'"
      exit 0
    fi
  fi
  sleep 0.1
done
`
	script := textutil.Apply(ctx, scriptTmpl, textutil.M{
		"Holder": holder,
	})
	_, _, err := SshExecBlock(ctx, host, script)
	return err
}

func (x SyncLock) WaitLockMany(ctx context.Context, holder string, hosts []string) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sort.Strings(hosts)

	ech := make(chan error, len(hosts))
	for _, host := range hosts {
		go func(host string) {
			ech <- x.WaitLock(ctx, holder, host)
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
