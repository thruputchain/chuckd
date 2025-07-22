LOOK AT:
- transmission reception latency
- drop rate
- worker pool latencies
- cpu utilization
- block profile

- while partialX is decoding, other methods are locked

[ ] confirm multiple connections to endpoint used uniformly
[ ] rpc parallel server responses
[ ] no worker queue


EXP

- small payloads

    // -- PARAMS --
    //
    pay-size: 300kb blobs (1000 txs)
    iterations: 100'000
    nodes: 3
    worker pools with queues
    //
    Endpoint_max_pending:       200,
    Endpoint_num_clients:       10,
    Endpoint_idle_ttl_msec:     60e3, // 1 minute
    Endpoint_redial_delay_msec: 100,  // 100 msec
    //
    Stream_receiver_worker_count: 64,
    Stream_sender_worker_count:   64,
    //
    Transmission_receiver_decode_worker_count: 32,
    Transmission_sender_worker_count:          16,
    //
    Num_payload_sheaves:   5,
    Num_parity_sheaves:    2,
    Max_chunk_bytes:       60e3, // 60K
    Partial_max_ttl_msec:  10e3, // 10 sec
    Partial_idle_ttl_msec: 1e3,  // 1 sec
    //
    // -- RESULTS --
    cpu-utilization: 40%-50%
    cpu-cache-misses: 24% (perf stat)
    throughput: ~2GB/sec (1.2GB/sec receive, 0.7GB/sec send)
    reception: 100%
    latency: 8ms
