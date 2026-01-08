#!/bin/sh

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))
ROOT=$(pwd)

# Clone only if ./scp isn't already a git repo
if [ -d scp/.git ]; then
  : "scp repo already present; reusing"
else
  # TODO: This clones the whole history, but there is no way to avoid this if targetting a specific commit.
  # In the future, it is possible to shallow clone targetting a branch!
  git clone --no-checkout git@github.com:groundhogchain/scp.git --quiet --no-progress scp
# git clone --depth 1 git@github.com:groundhogchain/scp.git --quiet --no-progress scp
fi

cd scp

want_commit="$(cat "${SCRIPT_DIR}/../scp.commit")"

# Try checkout first. If it fails, fetch/pull, then retry.
if git checkout -q "$want_commit"; then
  : "checked out $want_commit"
else
  git fetch --depth 1 origin "$want_commit"
  git checkout "$want_commit"
  # Ensure the commit exists locally, then retry checkout.
  # (This works even if the commit isn't in the shallow clone yet.)
  # git fetch --quiet --no-tags origin || git pull --quiet --ff-only
  # git checkout -q "$want_commit"
fi



cargo --config net.git-fetch-with-cli=true fetch --manifest-path Cargo.toml
cargo build --offline --locked -p scp_server --bin scpd
cp ./target/debug/scpd ${ROOT}/scpd.dev

cd ${ROOT}
