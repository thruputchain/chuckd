#!/bin/bash

set -xeuo pipefail

ROOT=$(pwd)

# scpd comes from the thruput-scp-server crate on crates.io
# (https://github.com/thruputchain/scp). Bump this with the scp release.
SCPD_VERSION=2.0.0
# The crate's rust-version. A bare `cargo install` resolves to the host's
# default toolchain, which can be older (the hog hosts' is 1.91.1), so pin it
# whenever rustup manages cargo. Bump this with SCPD_VERSION.
SCPD_RUST=1.96.0

# The install root and target directory live in the checkout, so repeated
# builds reuse compiled dependencies and nothing lands in ~/.cargo.
install_root="${ROOT}/.scpd"
install_args=(--locked --force --version "${SCPD_VERSION}" --root "${install_root}" --target-dir "${install_root}/target")

case "${SCPD_PROFILE-dev}" in
  dev) install_args+=(--debug) ;;
  release) ;;
  *) echo "invalid SCPD_PROFILE: ${SCPD_PROFILE}; expected dev or release" >&2; exit 2 ;;
esac

# jemalloc heap profiling is opt-in. Only the coordinator release build sets
# SCPD_HEAP_PROFILING=1; benches and hog-cluster builds must leave it unset.
case "${SCPD_HEAP_PROFILING-}" in
  ""|0) ;;
  1) install_args+=(--features heap-profiling) ;;
  *) echo "invalid SCPD_HEAP_PROFILING: ${SCPD_HEAP_PROFILING}; expected 1, 0 or unset" >&2; exit 2 ;;
esac

cargo=(cargo)
if command -v rustup >/dev/null 2>&1; then
  rustup toolchain install "${SCPD_RUST}" --profile minimal --no-self-update
  cargo+=("+${SCPD_RUST}")
fi

"${cargo[@]}" install "${install_args[@]}" thruput-scp-server
cp "${install_root}/bin/scpd" "${ROOT}/scpd.dev"
