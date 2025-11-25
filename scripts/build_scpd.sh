#!/bin/sh

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))
ROOT=$(pwd)

git clone --depth 1 git@github.com:groundhogchain/scp.git

cd scp
git checkout -q $(cat ${SCRIPT_DIR}/../scp.commit)

# build debug version of scpd
. ~/.cargo/env
CARGO_NET_GIT_FETCH_WITH_CLI=true cargo build -p scp_server --bin scpd
cp ./target/debug/scpd ${ROOT}/scpd.dev

cd ${ROOT}
