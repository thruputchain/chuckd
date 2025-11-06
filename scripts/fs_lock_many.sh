#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 PATH..." >&2
  exit 1
fi

# Collect arguments, sort lexicographically, and deduplicate
sorted_paths=()
while IFS= read -r path; do
  # Skip empty lines defensively
  [ -n "$path" ] || continue
  sorted_paths+=( "$path" )
done < <(printf '%s\n' "$@" | sort -u)

# Acquire a lock for each path in lexicographic order
for p in "${sorted_paths[@]}"; do
  lockdir="${p}.lockdir"

  # Busy-wait until we can create the lock directory (advisory lock)
  while ! mkdir "$lockdir" 2>/dev/null; do
    # Another process holds the lock; wait a bit and retry
    sleep 0.1
  done

  # Optionally record owner PID (not required for correctness)
  # echo "$$" > "${lockdir}/pid"
done

# All locks acquired
exit 0
