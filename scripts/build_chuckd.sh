#!/bin/bash

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))

ROOT=$(pwd)
cd ${SCRIPT_DIR}/../chuckd
build_args=()
if [[ -n "${CHUCKD_DISTRIBUTION_VERSION:-}" ]]; then
  if [[ ! "$CHUCKD_DISTRIBUTION_VERSION" =~ ^[0-9][0-9A-Za-z.-]*$ ]]; then
    echo "invalid CHUCKD_DISTRIBUTION_VERSION: $CHUCKD_DISTRIBUTION_VERSION" >&2
    exit 2
  fi
  build_args+=(
    -ldflags
    "-X github.com/thruputchain/chuckd.distributionVersion=$CHUCKD_DISTRIBUTION_VERSION"
  )
fi
go build "${build_args[@]}"
mv ./chuckd $ROOT/chuckd.dev
cd $ROOT
