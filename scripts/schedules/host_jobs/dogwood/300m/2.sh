#!/bin/sh
# chmod +x dogwood-bench-host=${HOG_HOST_2},seq=1.sh

set -xe


if [ $# -ne 1 ]; then
  echo "Usage: $0 <chuckd_path>"
  exit 1
fi


cat > dogwood-bench-config-host=${HOG_HOST_2},seq=1.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": {"scheme": "Ed25519", "ed25519": "6x9z2apK1Ku3UoPQTWNbBwBnBqZwq6djLoiduAwWoEUF"},
                "address": "${HOG_HOST_2}:4000"
            },
            "secret_key": {"scheme": "Ed25519", "ed25519": "6e0f53265180fd4c80741f4ad419599b21757ea1a0e233885d5e10d79d727efb586b8958193dd81237bc09d4fda7a9e11b1638c775871d1d9d9980108e628700"}
        },
        "known_validators": [
            {
                "public_key": {"scheme": "Ed25519", "ed25519": "3d1JgmibANYWwp3DmbX7avfvkjevAYpwZkN344xn2RFy"},
                "address": "${HOG_HOST_1}:4000"
            },
            {
                "public_key": {"scheme": "Ed25519", "ed25519": "6x9z2apK1Ku3UoPQTWNbBwBnBqZwq6djLoiduAwWoEUF"},
                "address": "${HOG_HOST_2}:4000"
            },
            {
                "public_key": {"scheme": "Ed25519", "ed25519": "6AvmTxLoyrp4hjyd4irz7YkNvdzjC8jNTrCJA4J23Dx4"},
                "address": "${HOG_HOST_3}:4000"
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
                    "ed25519": "3d1JgmibANYWwp3DmbX7avfvkjevAYpwZkN344xn2RFy"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "6x9z2apK1Ku3UoPQTWNbBwBnBqZwq6djLoiduAwWoEUF"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "6AvmTxLoyrp4hjyd4irz7YkNvdzjC8jNTrCJA4J23Dx4"
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
	--config dogwood-bench-config-host=${HOG_HOST_2},seq=1.json \
	 \
	 \
	 \
	 \
	 \
	--instance node-1 \
	--seed 3261433209860043341 \
	--metrics :2222 \
	bench dogwood \
	--addr :4000 \
	--bytes-per-pay 300000000 \
	--num-iterations 200 \
	 2> dogwood-bench-host=${HOG_HOST_2},seq=1.log | tee dogwood-bench-host=${HOG_HOST_2},seq=1.report
