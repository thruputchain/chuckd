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
                "public_key": "cf570ca0095853d21adcce9798b4f585756fce7f9aa4301ca0916ed08f8248f5",
                "address": "${HOG_HOST_2}:4000"
            },
            "secret_key": "2c630c25910887ce648417208afc619d86f61b4c4a8e114d922f6eea67f772b5cf570ca0095853d21adcce9798b4f585756fce7f9aa4301ca0916ed08f8248f5"
        },
        "known_validators": [
            {
                "public_key": "56ba670c40af9594a7cd8243d6aa97c4266a17e30ffc0f82c3e7990bda314c84",
                "address": "${HOG_HOST_1}:4000"
            },
            {
                "public_key": "cf570ca0095853d21adcce9798b4f585756fce7f9aa4301ca0916ed08f8248f5",
                "address": "${HOG_HOST_2}:4000"
            },
            {
                "public_key": "8a53342f221abfb9769a48ab199d1c584718cfac2f4e6ae33322bbb8b00dfa09",
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
                    "ed25519": "56ba670c40af9594a7cd8243d6aa97c4266a17e30ffc0f82c3e7990bda314c84"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "cf570ca0095853d21adcce9798b4f585756fce7f9aa4301ca0916ed08f8248f5"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "8a53342f221abfb9769a48ab199d1c584718cfac2f4e6ae33322bbb8b00dfa09"
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
	--seed 3107222966708712660 \
	--metrics :2222 \
	bench dogwood \
	--addr :4000 \
	--bytes-per-pay 30000000 \
	--num-iterations 1000 \
	 2> dogwood-bench-host=${HOG_HOST_2},seq=1.log | tee dogwood-bench-host=${HOG_HOST_2},seq=1.report
