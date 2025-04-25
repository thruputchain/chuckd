#!/bin/sh
set -e

/app/chuckd --config=$CONFIG_FILE --instance $INSTANCE --seed $SEED \
    solo start \
    --addr 0.0.0.0:9111
