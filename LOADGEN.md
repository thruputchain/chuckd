# Loadgen

## Build

To deploy a Dogwood node:

1. Set up your Go and Git environment to [work with private Go repositories](https://fueled.com/blog/using-private-modules-in-go/).

2. Install the `chuckd` binary:

```sh
go install github.com/groundhogchain/chuckd@latest
```

## Deploy

1. Start the hogd `loadgen_server` first
2. Start the chuckd loadgen server (this repo) with:

```sh
chuckd \
    --stack \       # stacktrace on error
    --level info \  # log level info
    bench loadgen \
        --in-addr :7888 \   # bind address for (this) chuckd loadgen server
        --loadgen-addr localhost:11222 \    # address of hogd loadgen server
        --num-accts 10000 \     # number of accounts
        --num-txs 1000000 \     # number of txs per block
        --exec-batch 100           # number of txs per call to exec
        --exec-par 10 \              # number of conns per exec
```

3. Start all executors
4. Start all db shards
