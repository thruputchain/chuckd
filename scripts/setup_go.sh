# Keeps the Go module cache inside the checkout, which the benchmark schedules
# source so that each hog_sched job starts from a clean one. The module and
# its dependencies are public, so the toolchain uses the default proxy and
# checksum database.
export GOPATH=$(pwd)/__go__
export PATH=$PATH:$GOPATH/bin
