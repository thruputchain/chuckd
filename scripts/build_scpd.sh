#!/bin/sh

set -ex pipefile

SCRIPT_DIR=$(dirname $(readlink -f "$0"))

git clone --depth 1 git@github.com:groundhogchain/scp.git

cd scp
git checkout $(cat ${SCRIPT_DIR}/../scp.commit)

# build debug version of scpd
. ~/.cargo/env
cargo build -p scp_server --bin scpd
cp ./target/debug/scpd ../
