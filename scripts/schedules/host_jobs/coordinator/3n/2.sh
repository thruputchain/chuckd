#!/bin/sh
set -e

if [ $# -ne 2 ]; then
  echo "Usage: $0 <chuckd_path> <scpd_path>"
  exit 1
fi

cat > config-${HOG_HOST_2}.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "982951c7de7f91bde1770134b6f301b7e4b8a4f4ce250a1ce5f9fd7ecde52bd7",
                "address": "${HOG_HOST_2}:50000"
            },
            "secret_key": "abb5c71b2296c1d0e4925ccf210bcdb20665dea5e02a2eab3a593efdaa20d021982951c7de7f91bde1770134b6f301b7e4b8a4f4ce250a1ce5f9fd7ecde52bd7"
        },
        "known_validators": [
            {
                "public_key": "970c9ba081452fa7f094bbba51966f4a64c9108ea6fec6d1dcbbad38e25e0f56",
                "address": "${HOG_HOST_1}:50000"
            },
            {
                "public_key": "982951c7de7f91bde1770134b6f301b7e4b8a4f4ce250a1ce5f9fd7ecde52bd7",
                "address": "${HOG_HOST_2}:50000"
            },
            {
                "public_key": "24454aa846034d3d8bb9110797c97132cb63f91794db548d9a6ff03249eea12e",
                "address": "${HOG_HOST_3}:50000"
            }
        ]
    },
    "discovery_config": {
        "advertise_interval_msec": "15000",
        "advertise_par": 5,
        "snapshot_ttl_msec": "21000"
    },
    "router_config": {
        "tree_fan_out": 5
    },
    "dogwood_config": {
        "route_count": 50,
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
    "mempool_config": {
        "transaction_ttl_msec": "6000",
        "submission_lot_size": "1000",
        "submission_lot_wait_msec": "50"
    },
    "catchup_config": {
        "catchup_block_ttl_sec": "15",
        "catchup_block_age": "5"
    },
    "scp_config": {
        "max_timeout_ms": "1000",
        "initial_ballot_timeout_ms": "750",
        "increment_ballot_timeout_ms": "750",
        "initial_nomination_timeout_ms": "750",
        "increment_nomination_timeout_ms": "750",
        "quorum_set": {
            "threshold": 2,
            "validators": [
                {
                    "scheme": "Ed25519",
                    "ed25519": "970c9ba081452fa7f094bbba51966f4a64c9108ea6fec6d1dcbbad38e25e0f56"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "982951c7de7f91bde1770134b6f301b7e4b8a4f4ce250a1ce5f9fd7ecde52bd7"
                },
                {
                    "scheme": "Ed25519",
                    "ed25519": "24454aa846034d3d8bb9110797c97132cb63f91794db548d9a6ff03249eea12e"
                }
            ],
            "inner_sets": [
            ]
        }
    },
    "baton_config": {
        "skew_msec": "40",
        "heartbeat_msec": "600",
        "patience_msec": "3000",
        "max_catchup_height": 10000
    },
    "repair_config": {
        "base_delay_msec": "200",
        "max_delay_msec": "5000",
        "multiplier_nom": 3,
        "multiplier_denom": 2
    },
    "storage_config": {
        "reclaim_interval_msec": "300000",
        "max_catchup_blocks": 86400,
        "max_scp_slots": 86400
    }
}
EOF
rm -rf ./validator-storage-${HOG_HOST_2}
$1 -l ${LOG_LEVEL} --stack --config config-${HOG_HOST_2}.json \
    --rpc-log-stats \
	--instance 982951 \
	--seed 668639415096168636 \
	--metrics :2223 \
		bench coordinator \
            --debug-port 5500 \
			--txs-per-sec 1000 \
            --num-commits 50 \
			--storage-dir ./validator-storage-${HOG_HOST_2} \
			--out-addr :50000 \
			--scpd-path $2 \
			--scp-handle-addr 127.0.0.1:3300 \
			--scp-control-addr 127.0.0.1:4400 \
			--scp-is-validator true \
                > >(tee validator-${HOG_HOST_2}.report) \
                2> >(tee validator-${HOG_HOST_2}.log >&2)
