#!/bin/sh
set -e

docker run \
  -e INSTANCE=$INSTANCE \
  -e SEED=$SEED \
  -v $CONFIG:/app/config/config.json \
  groundhogchain:chuckd_solo.latest \
  -v
