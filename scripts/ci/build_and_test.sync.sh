#!/usr/bin/env bash
set -euox pipefail

SCRIPTS_CI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# SCRIPTS_DIR="$(dirname "$SCRIPTS_CI_DIR")"

source flag_job.sh
parse_flag_j "$@"

# lock machines
host=hog6
hog_hold_lock --j=$job --host=$host &
hog_wait_lock --j=$job --host=$host

# build on $host
hog_exec --j=$job --host=$host < "$SCRIPTS_CI_DIR"/build.sh
hog_exec --j=$job --host=$host < "$SCRIPTS_CI_DIR"/test.sh

# copy output artifacts here
hog_copy_from --j=$job --from=$host --in=chuckd/coverage.out --out=coverage.out
