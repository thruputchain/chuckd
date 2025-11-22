#!/bin/sh
# chmod +x dogwood-bench-host=hog6,seq=0.sh

set -xe


if [ $# -ne 1 ]; then
  echo "Usage: $0 <chuckd_path>"
  exit 1
fi


cat > dogwood-bench-config-host=hog6,seq=0.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "26f0a0b4e6429136e5c2a4f060c83bc44e955f717862e740ba2b7a79e6c16c5c",
                "address": "hog6:4000"
            },
            "secret_key": "47a0ab57c1d41d7e29f223ee5c6740da7f9e8b4ac3dab64127ba9bec44ae38fd26f0a0b4e6429136e5c2a4f060c83bc44e955f717862e740ba2b7a79e6c16c5c"
        },
        "known_validators": [
            {
                "public_key": "26f0a0b4e6429136e5c2a4f060c83bc44e955f717862e740ba2b7a79e6c16c5c",
                "address": "hog6:4000"
            },
            {
                "public_key": "586b8958193dd81237bc09d4fda7a9e11b1638c775871d1d9d9980108e628700",
                "address": "hog7:4000"
            },
            {
                "public_key": "4cd574bf0028b118f15e37c944e72f4b6f021fbf9043b7cfe5a88040b9347f69",
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
                    "ed25519": "26f0a0b4e6429136e5c2a4f060c83bc44e955f717862e740ba2b7a79e6c16c5c"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "586b8958193dd81237bc09d4fda7a9e11b1638c775871d1d9d9980108e628700"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "4cd574bf0028b118f15e37c944e72f4b6f021fbf9043b7cfe5a88040b9347f69"
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

perf stat -e cache-misses,cache-references,L1-dcache-loads,L1-dcache-load-misses,L1-icache-load-misses,LLC-loads,LLC-load-misses,branch-misses,page-faults \
	$1 \
	-l info \
	--config dogwood-bench-config-host=hog6,seq=0.json \
	 \
	 \
	 \
	 \
	 \
	--instance node-0 \
	--seed 2056001763466157503 \
	--metrics :2222 \
	bench dogwood \
	--addr :4000 \
	--bytes-per-pay 300000000 \
	--num-iterations 200 \
	 2> dogwood-bench-host=hog6,seq=0.log | tee dogwood-bench-host=hog6,seq=0.report
