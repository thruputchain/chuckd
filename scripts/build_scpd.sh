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

cd ${SCRIPT_DIR}/../scp

cargo --config net.git-fetch-with-cli=true fetch --manifest-path Cargo.toml
cargo build "${build_args[@]}" -p scp_server --bin scpd
cp "./target/${target_profile}/scpd" "${ROOT}/scpd.dev"

cd ${ROOT}
