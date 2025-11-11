#!/usr/bin/env bash
set -euox pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source flag_job.sh
parse_flag_j "$@"

# clone repo
git clone --depth 1 git@github.com:groundhogchain/chuckd.git

# lock machines
host_1=hog7
host_2=hog8

hog_hold_lock --j=$job --host=$host_1 --host=$host_2 &
hog_wait_lock --j=$job --host=$host_1 --host=$host_2

# build
build_script="$(< "$SCRIPT_DIR"/clone_and_build.sh)"

hog_par <<EOF
    hog_exec --j=$job --host=$host_1 <<< "$build_script"
    hog_exec --j=$job --host=$host_2 <<< "$build_script"
EOF

# benchmark
bench_script_1="$(< \
    sed -e "s/{{LABEL}}/300k/g" \
        -e "s/{{HOST}}/${host_1}/g" \
        -e "s/{{NODE}}/node-1-of-2/g" \
        "$SCRIPT_DIR"/bench.tmpl.sh)"

bench_script_2="$(< \
    sed -e "s/{{LABEL}}/300k/g" \
        -e "s/{{HOST}}/${host_2}/g" \
        -e "s/{{NODE}}/node-2-of-2/g" \
        "$SCRIPT_DIR"/bench.tmpl.sh)"

hog_par <<EOF
    hog_exec --j=$job --host=$host_1 <<< "$bench_script_1"
    hog_exec --j=$job --host=$host_2 <<< "$bench_script_2"
EOF
