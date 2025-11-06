#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 PATH..." >&2
  exit 1
fi

# Collect arguments, sort lexicographically, and deduplicate
sorted_paths=()
while IFS= read -r path; do
  [ -n "$path" ] || continue
  sorted_paths+=( "$path" )
done < <(printf '%s\n' "$@" | sort -u)

# Release the locks for all paths
for p in "${sorted_paths[@]}"; do
  lockdir="${p}.lockdir"

  if [ -d "$lockdir" ]; then
    # Remove any metadata like pid file if you added one:
    # rm -f "${lockdir}/pid" 2>/dev/null || true

    if ! rmdir "$lockdir" 2>/dev/null; then
      echo "Warning: failed to remove lock for '$p' (dir: '$lockdir')" >&2
    fi
  fi
done

exit 0
