#!/bin/bash

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))
ROOT=$(pwd)

cd ${SCRIPT_DIR}/../scp

cargo --config net.git-fetch-with-cli=true fetch --manifest-path Cargo.toml
cargo build --offline --locked -p scp_server --bin scpd
cp ./target/debug/scpd ${ROOT}/scpd.dev

cd ${ROOT}
