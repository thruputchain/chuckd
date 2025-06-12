#!/bin/sh
set -x

killall chuckd
go install ../chuckd

# proposing and validating speeds are half of what they would be in actual load generation

# mac: 14x arm64, 4 GHz
# hog: 64x2x amd64, 3 GHz

# 14 exec

# mac: generating 560K tx/sec, proposing 340K tx/sec, validating 270K tx/sec
# hog: generating 450K tx/sec, proposing 330K tx/sec, validating 246K tx/sec
# 100 batches of 1000 txs per exec
chuckd -v bench loadgen-test --gen-par 14 --num-exec 14 --exec-par 1 --exec-batch 1000 --tpb 1400000

# 64 exec

# hog: generating 1.5M tx/sec, proposing 300K tx/sec, validating 210K tx/sec
# 21 batches of 1000 txs per exec
chuckd -v bench loadgen-test --gen-par 64 --num-exec 64 --exec-par 1 --exec-batch 1000 --tpb 1400000

# hog: generating 1.5M tx/sec, proposing 255K tx/sec, validating 184K tx/sec
# 218 batches of 100 txs per exec
chuckd -v bench loadgen-test --gen-par 64 --num-exec 64 --exec-par 1 --exec-batch 100 --tpb 1400000

# 128 exec

# hog: generating 1.9M tx/sec, proposing 307K tx/sec, validating 199 tx/sec
# 11 batches of 1000 txs per exec
chuckd -v bench loadgen-test --gen-par 128 --num-exec 128 --exec-par 1 --exec-batch 2000 --tpb 1400000

# hog: generating 2M tx/sec, proposing 306K tx/sec, validating 190K tx/sec
# 11 batches of 1000 txs per exec
chuckd -v bench loadgen-test --gen-par 128 --num-exec 128 --exec-par 1 --exec-batch 1000 --tpb 1400000

# hog: generating 1.9M tx/sec, proposing 243K tx/sec, validating 172K tx/sec
# 109 batches of 100 txs per exec
chuckd -v bench loadgen-test --gen-par 128 --num-exec 128 --exec-par 1 --exec-batch 100 --tpb 1400000
