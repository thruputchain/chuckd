#!/usr/bin/env bash
trap 'kill 0' EXIT
set -euox pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$SCRIPT_DIR"/flag_job.sh
parse_flag_j "$@"

# lock machines
host=hog6
hog_hold_lock --j=$job --host=$host &
hog_wait_lock --j=$job --host=$host

# build on $host
hog_exec --j=$job --host=$host < "$SCRIPT_DIR"/clone_and_build.sh
hog_exec --j=$job --host=$host < "$SCRIPT_DIR"/test.sh

# copy output artifacts here
hog_copy_from --j=$job --from=$host --in=chuckd/coverage.out --out=coverage.out
