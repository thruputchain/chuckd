#!/bin/sh
# chmod +x dogwood-bench-host=${HOG_HOST_2},seq=1.sh

if [ $# -ne 1 ]; then
  echo "Usage: $0 <chuckd_path>"
  exit 1
fi

set -xe

cat > dogwood-bench-config-host=${HOG_HOST_2},seq=1.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": {"scheme": "Ed25519", "ed25519": "CFVg7SVpsQDDqia668sbtiu5L9M34P5JFm39bPFUtxgK"},
                "address": "${HOG_HOST_2}:4000"
            },
            "secret_key": {"scheme": "Ed25519", "ed25519": "3061009c1fb320376b6c80dc50d054148cc5e26d0d02222f3b793327fd07f5b6a7271c142fe68384d9543ff85ee8614e7ddd3b60db8d80e7bf38f534193fbb8c"}
        },
        "known_validators": [
            {
                "public_key": {"scheme": "Ed25519", "ed25519": "GhaZ5bWgDDymAapwjTX29EBSVt5X7pCbBQVsZLV1Nf5L"},
                "address": "${HOG_HOST_1}:4000"
            },
            {
                "public_key": {"scheme": "Ed25519", "ed25519": "CFVg7SVpsQDDqia668sbtiu5L9M34P5JFm39bPFUtxgK"},
                "address": "${HOG_HOST_2}:4000"
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
                    "ed25519": "GhaZ5bWgDDymAapwjTX29EBSVt5X7pCbBQVsZLV1Nf5L"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "CFVg7SVpsQDDqia668sbtiu5L9M34P5JFm39bPFUtxgK"
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
	--seed 1432401676018881212 \
	--metrics :2222 \
	bench dogwood \
	--addr :4000 \
	--bytes-per-pay 300000 \
	--num-iterations 50000 \
	 2> dogwood-bench-host=${HOG_HOST_2},seq=1.log | tee dogwood-bench-host=${HOG_HOST_2},seq=1.report
