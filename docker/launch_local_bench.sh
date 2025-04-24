#!/bin/sh
set -e

docker network create --subnet=172.18.0.0/16 chuckd-bench-network

docker run --network chuckd-bench-network --ip 172.18.0.11 --name chuckd-bench-1 -d \
    -e INSTANCE=bench-1 \
    -e SEED=1000 \
    -v ./local_bench_config/config-1.json:/app/config/config.json \
    groundhogchain:chuckdbench.latest \
    -v \
    --fanout 5 \
    --iterations 3 \
    --payperchan 10 \
    --paysize 100

docker run --network chuckd-bench-network --ip 172.18.0.12 --name chuckd-bench-2 -d \
    -e INSTANCE=bench-2 \
    -e SEED=2000 \
    -v ./local_bench_config/config-2.json:/app/config/config.json \
    groundhogchain:chuckdbench.latest \
    -v \
    --fanout 5 \
    --iterations 3 \
    --payperchan 10 \
    --paysize 100

docker run --network chuckd-bench-network --ip 172.18.0.13 --name chuckd-bench-3 -d \
    -e INSTANCE=bench-3 \
    -e SEED=3000 \
    -v ./local_bench_config/config-3.json:/app/config/config.json \
    groundhogchain:chuckdbench.latest \
    -v \
    --fanout 5 \
    --iterations 3 \
    --payperchan 10 \
    --paysize 100


# docker network inspect chuckd-bench-network
