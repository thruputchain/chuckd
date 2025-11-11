#!/usr/bin/env bash
set -euox pipefail

cd chuckd

echo "cloning metrics history ..."

./scripts/clone_metrics_history.sh


echo "running benchmark ..."

source ./scripts/git_info.sh

# LABEL=300k
# HOST=hog7
# NODE=node-1-of-2

METRIC_NAME="dogwood-{{LABEL}}-{{NODE}}"

./metrics_history/scripts/make_metric_file.sh $METRIC_NAME $GIT_branch $GIT_commit $GIT_commit_time

./scripts/dogwood_bench/{{LABEL}}/{{HOST}}.sh ./chuckd/chuckd > \
    ./metrics_history/metrics/$METRIC_NAME/$GIT_branch/${GIT_commit_time}_$GIT_commit


echo "committing metrics ..."

./scripts/commit_metrics.sh
