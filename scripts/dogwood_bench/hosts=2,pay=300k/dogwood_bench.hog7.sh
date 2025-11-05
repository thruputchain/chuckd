#!/bin/sh
# chmod +x dogwood-bench-host=hog7,seq=0.sh

set -xe

cat > dogwood-bench-config-host=hog7,seq=0.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "e727a1ae362cdc6a2142114f5cfaa26332100688a929d04c260628c2f68697e6",
                "address": "hog7:4000"
            },
            "secret_key": "2ec3a8f18ce1b7d8602f517c3029e729192cbe62a9f32ffe870ef627d5aecabee727a1ae362cdc6a2142114f5cfaa26332100688a929d04c260628c2f68697e6"
        },
        "known_validators": [
            {
                "public_key": "e727a1ae362cdc6a2142114f5cfaa26332100688a929d04c260628c2f68697e6",
                "address": "hog7:4000"
            },
            {
                "public_key": "8235ce8df3e2d8d6ab7a5cefb277a731db9f8213e13de0cdd7cef6b77943c203",
                "address": "hog8:4000"
            }
        ]
    },
    "wallet_config": null,
    "discovery_config": null,
    "router_config": {
        "tree_fan_out": 5
    },
    "dogwood_config": {
        "route_count": 17,
        "route_time_to_live_msec": "60000",
        "route_max_latency_msec": "1000",
        "endpoint_max_pending": 800,
        "endpoint_num_clients": 16,
        "endpoint_idle_ttl_msec": "60000",
        "endpoint_dial_timeout_msec": "2000",
        "endpoint_redial_delay_msec": "100",
        "endpoint_call_timeout_msec": "2000",
        "stream_receiver_worker_count": 512,
        "stream_sender_worker_count": 32,
        "transmission_receiver_decode_worker_count": 512,
        "transmission_sender_worker_count": 32,
        "transmission_partial_max_ttl_msec": "10000",
        "transmission_partial_idle_ttl_msec": "1000"
    },
    "mempool_config": null,
    "catchup_config": null,
    "scp_config": {
        "max_timeout_ms": "0",
        "initial_ballot_timeout_ms": "0",
        "increment_ballot_timeout_ms": "0",
        "initial_nomination_timeout_ms": "0",
        "increment_nomination_timeout_ms": "0",
        "quorum_set": {
            "threshold": 2,
            "validators": [
                {
                    "scheme": "Ed25519",
                    "ed25519": "e727a1ae362cdc6a2142114f5cfaa26332100688a929d04c260628c2f68697e6"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "8235ce8df3e2d8d6ab7a5cefb277a731db9f8213e13de0cdd7cef6b77943c203"
                }
            ],
            "inner_sets": [
            ]
        }
    },
    "baton_config": null,
    "repair_config": null,
    "storage_config": null
}
EOF

echo "benchmark working directory $(pwd)"

perf stat -e cache-misses,cache-references,L1-dcache-loads,L1-dcache-load-misses,L1-icache-load-misses,LLC-loads,LLC-load-misses,branch-misses,page-faults chuckd -l info --config dogwood-bench-config-host=hog7,seq=0.json \
	 \
	 \
	 \
	 \
	 \
	--instance node-0 \
	--seed 233861746498809834 \
	--metrics :2222 \
	bench dogwood \
	--addr :4000 \
	--bytes-per-pay 300000 \
	--num-iterations 50000 \
	 2> dogwood-bench-host=hog7,seq=0.log | tee dogwood-bench-host=hog7,seq=0.report
