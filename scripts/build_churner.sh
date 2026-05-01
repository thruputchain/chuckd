#!/bin/bash

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))
ROOT=$(pwd)

cd ${SCRIPT_DIR}/../hogx

cargo --config net.git-fetch-with-cli=true fetch --manifest-path Cargo.toml
cargo build --offline --locked -p churner --bin hogx-churner
cp ./target/debug/hogx-churner ${ROOT}/hogx-churner

cd ${ROOT}
