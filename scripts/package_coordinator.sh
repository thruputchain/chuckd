#!/bin/bash

set -euo pipefail

case "${1:-}" in
  x86_64|arm64) architecture=$1 ;;
  *) echo "expected coordinator architecture: x86_64 or arm64" >&2; exit 2 ;;
esac

package_dir="chuckd-coordinator-linux-${architecture}"
# Keep debuginfo outside packaging's chuckd-coordinator-linux-*.tar.gz glob.
debug_dir="chuckd-coordinator-debuginfo-linux-${architecture}"
mkdir -p "$package_dir" "$debug_dir/usr/lib/debug/.build-id"

for binary in chuckd scpd; do
  install -m 0755 "${binary}.dev" "$package_dir/$binary"
  build_id=$(readelf -n "$package_dir/$binary" | awk '/Build ID:/ {print $3}')
  if [[ -z "$build_id" && "$binary" == chuckd ]]; then
    # Go may have only a Go build-id. Keep a named fallback without linker changes.
    debug_file="$debug_dir/usr/lib/debug/chuckd.debug"
  elif [[ "$build_id" =~ ^([0-9a-f]{2}){2,}$ ]]; then
    debug_file="$debug_dir/usr/lib/debug/.build-id/${build_id:0:2}/${build_id:2}.debug"
  else
    echo "missing or invalid GNU build-id for $binary: $build_id" >&2
    exit 1
  fi
  # Refuse collisions rather than overwriting another binary's debug information.
  test ! -e "$debug_file"
  mkdir -p "$(dirname "$debug_file")"
  objcopy --only-keep-debug "$package_dir/$binary" "$debug_file"
  # strip-debug preserves .symtab; never use strip-all or strip-unneeded.
  objcopy --strip-debug --add-gnu-debuglink="$debug_file" "$package_dir/$binary"
  readelf -S "$package_dir/$binary" | awk '
    /\.symtab/ {found=1}
    END {exit !found}
  '
done

tar -C "$package_dir" -czf "${package_dir}.tar.gz" chuckd scpd
tar -C "$debug_dir" -czf "${debug_dir}.tar.gz" usr/lib/debug
