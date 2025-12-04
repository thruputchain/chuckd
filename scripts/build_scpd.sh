#!/bin/sh

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))
ROOT=$(pwd)

git clone --depth 1 git@github.com:groundhogchain/scp.git --quiet --no-progress

# build debug version of scpd
cd scp

git config core.sshCommand "ssh -F $HOME/.ssh/config"

git checkout -q $(cat ${SCRIPT_DIR}/../scp.commit)
cargo --config net.git-fetch-with-cli=true fetch --manifest-path Cargo.toml
cargo build --offline --locked -p scp_server --bin scpd
cp ./target/debug/scpd ${ROOT}/scpd.dev

cd ${ROOT}
