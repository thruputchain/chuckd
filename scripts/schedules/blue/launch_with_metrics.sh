if [[ ${NODE} == *"1-of"* ]]; then
    capture_metrics=true
else
    capture_metrics=false
fi

# NOTE: override capture_metrics as it is causing ci to return non-zero exit code,
# even when the experiment succeeds.
capture_metrics=false

if [[ $capture_metrics == true ]]; then

    echo "cloning metrics history ..."
    ./scripts/clone_metrics_history.sh

    echo "running chuckd with metric capture ..."

    source ./scripts/git_info.sh

    # e.g. METRIC=dogwood, LABEL=300k, HOST=hog7, NODE=node-1-of-2
    METRIC_NAME="${METRIC}-${LABEL}"

    ./metrics_history/scripts/make_metric_file.sh $METRIC_NAME $GIT_branch $GIT_commit $GIT_commit_time

    "${HOST_JOBS_DIR}"/"${METRIC}"/"${LABEL}"/"${HOST}".sh "$@" | \
        tee ./metrics_history/metrics/$METRIC_NAME/$GIT_branch/${GIT_commit_time}_$GIT_commit

    echo "committing metrics ..."
    ./scripts/commit_metrics.sh

else

    echo "running chuckd (without metric capture) ..."

    "${HOST_JOBS_DIR}"/"${METRIC}"/"${LABEL}"/"${HOST}".sh "$@"

fi
