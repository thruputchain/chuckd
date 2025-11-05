#!/bin/sh
# chmod +x dogwood-bench-id:0-host:hog7.sh

set -xe

cat > dogwood-bench-config-hog7.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "d2c9b46afa6da0f21a1a04ac7bbc2157cbab1f81d82934e2ebde474fe5ce8caf",
                "address": "hog7:4000"
            },
            "secret_key": "c681d2d5259ed657ca4a38de41ffca7956f3d9d2244433c7837790ed58ae9e2ed2c9b46afa6da0f21a1a04ac7bbc2157cbab1f81d82934e2ebde474fe5ce8caf"
        },
        "known_validators": [
            {
                "public_key": "d2c9b46afa6da0f21a1a04ac7bbc2157cbab1f81d82934e2ebde474fe5ce8caf",
                "address": "hog7:4000"
            },
            {
                "public_key": "c8aa3d73d24c0644dc0f8ccb80f3d20f289277f4d1bbff3a84513f0873660b80",
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
                    "ed25519": "d2c9b46afa6da0f21a1a04ac7bbc2157cbab1f81d82934e2ebde474fe5ce8caf"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "c8aa3d73d24c0644dc0f8ccb80f3d20f289277f4d1bbff3a84513f0873660b80"
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

perf stat -e cache-misses,cache-references,L1-dcache-loads,L1-dcache-load-misses,L1-icache-load-misses,LLC-loads,LLC-load-misses,branch-misses,page-faults chuckd -l info --config dogwood-bench-config-hog7.json \
	 \
	 \
	 \
	 \
	 \
	--instance node-0 \
	--seed 1156803210893542008 \
	--metrics :2222 \
	bench dogwood \
	--addr :4000 \
	--bytes-per-pay 300000 \
	--num-iterations 50000 \
	> dogwood-bench-host:hog7-id:0.log 2>&1 & 2> dogwood-bench-hog7.log | tee dogwood-bench-hog7.report
