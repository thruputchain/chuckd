#!/bin/sh
set -e

/app/chuckd --config=$CONFIG_FILE --instance $INSTANCE --seed $SEED \
    bench broadcast \
    --addr 0.0.0.0:8181

# pass these flags on the docker command line
# --fanout XXX --iterations XXX --payperchan XXX --paysize XXX
