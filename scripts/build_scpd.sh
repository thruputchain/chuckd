#!/bin/sh

set -x
git clone --depth 1 git@github.com:groundhogchain/scp.git

cd scp
git checkout $(cat ../scp.commit)

# build debug version of scpd
cargo build -p scp_server --bin scpd
cp ./target/debug/scpd ../
