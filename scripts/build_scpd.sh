#!/bin/bash

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))
ROOT=$(pwd)
build_args=(--offline --locked)

case "${SCPD_PROFILE-dev}" in
  dev) target_profile=debug ;;
  release) build_args+=(--release); target_profile=release ;;
  *) echo "invalid SCPD_PROFILE: ${SCPD_PROFILE}; expected dev or release" >&2; exit 2 ;;
esac

# jemalloc heap profiling is opt-in. Only the coordinator release build sets
# SCPD_HEAP_PROFILING=1; benches and hog-cluster builds must leave it unset.
case "${SCPD_HEAP_PROFILING-}" in
  ""|0) ;;
  1) build_args+=(--features heap-profiling) ;;
  *) echo "invalid SCPD_HEAP_PROFILING: ${SCPD_HEAP_PROFILING}; expected 1, 0 or unset" >&2; exit 2 ;;
esac

cd ${SCRIPT_DIR}/../scp

cargo --config net.git-fetch-with-cli=true fetch --manifest-path Cargo.toml
cargo build "${build_args[@]}" -p scp_server --bin scpd
cp "./target/${target_profile}/scpd" "${ROOT}/scpd.dev"

cd ${ROOT}
