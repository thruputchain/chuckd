#!/bin/sh
set -x

killall chuckd
go install -race ../chuckd

# chuckd deploy local-bench-testcluster > local-bench-testcluster.sh
# chmod +x local-bench-testcluster.sh
chmod +x local-bench-testcluster.sh

./local-bench-testcluster.sh
