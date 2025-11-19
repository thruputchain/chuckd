#!/bin/sh

set -xeuo pipefail

SCRIPT_DIR=$(dirname $(readlink -f "$0"))

ROOT=$(pwd)
cd ${SCRIPT_DIR}/../chuckd
go build
mv ./chuckd $ROOT/chuckd
cd $ROOT
