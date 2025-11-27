#!/bin/sh

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))
ROOT=$(pwd)

git clone --depth 1 git@github.com:groundhogchain/scp.git

echo $HOME
echo $GIT_CONFIG_GLOBAL
echo ~

GIT_TRACE=1 git ls-remote https://github.com/groundhogchain/protocol.git
# build debug version of scpd
. ~/.cargo/env

cd scp
git checkout -q $(cat ${SCRIPT_DIR}/../scp.commit)
cd ..

GIT_TRACE=1 git clone git@github.com:groundhogchain/protocol.git || true
git clone git@github.com:groundhogchain/protocol-rs.git || true
git clone git@github.com:groundhogchain/xdr_rpc.git || true
git clone git@github.com:groundhogchain/xdr_common.git || true
git clone git@github.com:groundhogchain/xdrs.git || true

cargo --config net.git-fetch-with-cli=true fetch --manifest-path scp/Cargo.toml

cd scp
cargo build --offline --locked -p scp_server --bin scpd
cp ./target/debug/scpd ${ROOT}/scpd.dev

cd ${ROOT}
