#!/bin/bash

set -xeuo pipefail

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
ROOT=$(pwd)

cd "${SCRIPT_DIR}/../hogx"

# Protocol bindings require Rust 1.96; keep the worker's default toolchain unchanged.
CHURNER_RUST_TOOLCHAIN=1.96.0
rustup toolchain install "$CHURNER_RUST_TOOLCHAIN" --profile minimal --no-self-update
cargo +"$CHURNER_RUST_TOOLCHAIN" --config net.git-fetch-with-cli=true fetch --manifest-path Cargo.toml
cargo +"$CHURNER_RUST_TOOLCHAIN" build --offline --locked -p churner --bin hogx-churner
cp ./target/debug/hogx-churner "${ROOT}/hogx-churner"

cd "${ROOT}"
