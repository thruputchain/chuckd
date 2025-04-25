#!/bin/sh
set -e

docker run \
  -e INSTANCE=$INSTANCE \
  -e SEED=$SEED \
  -v $CONFIG:/app/config/config.json \
  groundhogchain:chuckd_bench.latest \
  -v \
  --fanout 5 \
  --iterations 3 \
  --payperchan 10 \
  --paysize 100
