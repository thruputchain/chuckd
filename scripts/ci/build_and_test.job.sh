#!/usr/bin/env bash
set -euo pipefail

# SCRIPTS_CI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# SCRIPTS_DIR="$(dirname "$SCRIPTS_CI_DIR")"

source flag_job.sh
parse_flag_j "$@"

# lock machines
hog_hold_lock --j=id --host=hog6 &
hog_wait_lock --j=id --host=hog6

# build on hog6
hog_exec --j=$JOB --host=hog6 < build_and_test.sh

# copy output artifacts here
hog_copy_from --j=$JOB --from=hog6 --in=coverage.out --out=coverage.out
