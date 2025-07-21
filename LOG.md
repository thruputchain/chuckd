LOOK AT:
- worker pool latencies
- transmission reception latency
- drop rate
- cpu utilization



300kb blobs (1000 txs)
20 TCP connections between node pairs
60kb/rpc <— ???
7 multicast trees (5 payload + 2 parity)
28%-40% cpu utilization on all 128 cores
3 nodes
1GB/sec - 3GB/sec (650Mb/sec-1.5GB/sec receive, 400MB/sec-1.5GB/sec send)
100% reception within 1 sec (max latency)
20% cpu cache misses (perf stat)


30MB blobs (1 nomination 1M tx hashes)
20 TCP connections between node pairs
60kb/rpc <— ???
7 multicast trees (5 payload + 2 parity)
45% cpu utilization on all 128 cores
3 nodes
4GB/sec (3.2GB/sec receive, 1.2GB/sec send)
98% reception within 3 sec (max latency)
17% cpu cache misses (perf stat)


QUIC is too slow
