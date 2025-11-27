#!/bin/sh

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))
ROOT=$(pwd)

git clone --depth 1 git@github.com:groundhogchain/scp.git

# build debug version of scpd
. ~/.cargo/env

cd scp
git checkout -q $(cat ${SCRIPT_DIR}/../scp.commit)

cd ..
export GIT_CONFIG_GLOBAL="${ROOT}/.gitconfig"
export HOME="${ROOT}"
cargo --config net.git-fetch-with-cli=true fetch --manifest-path scp/Cargo.toml

cd scp
cargo build --offline --locked -p scp_server --bin scpd
cp ./target/debug/scpd ${ROOT}/scpd.dev

cd ${ROOT}
