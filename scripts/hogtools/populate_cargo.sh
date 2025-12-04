#!/bin/sh

set -ex

hosts=("hog1" "hog2" "hog3" "hog4" "hog5" "hog6" "hog7" "hog8" "hog9" "hog10" "hog11")

content='
set -ex
curl -sSf https://sh.rustup.rs | sh -s -- -y
source ~/.cargo/env
rustup default stable
'

for host in "${hosts[@]}"; do
echo "Installing cargo and rustup on $host..."
printf "%s" "$content" | ssh "$host" bash -s
done
