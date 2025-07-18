#!/bin/sh
set -e

cat > config-leader.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "lwyboIFFL6fwlLu6UZZvSmTJEI6m/sbR3LutOOJeD1Y=",
                "address": "localhost:60000"
            },
            "secret_key": "uctt4Stz8Zaa3ut8EQLZQg9pLaHJd83rFSccagkNBriXDJuggUUvp/CUu7pRlm9KZMkQjqb+xtHcu6044l4PVg=="
        },
        "quorum_set": {
            "threshold": 0,
            "validators": [
            ],
            "inner_sets": [
            ]
        },
        "known_validators": [
            {
                "public_key": "mClRx95/kb3hdwE0tvMBt+S4pPTOJQoc5fn9fs3lK9c=",
                "address": "localhost:50000"
            },
            {
                "public_key": "JEVKqEYDTT2LuREHl8lxMstj+ReU21SNmm/wMknuoS4=",
                "address": "localhost:50001"
            },
            {
                "public_key": "eJOCvsH4nfQSZdKTySLutdWojYpGHAbs8H5iAKzMXQA=",
                "address": "localhost:50002"
            }
        ]
    },
    "wallet_config": null,
    "discovery_config": null,
    "router_config": null,
    "broadcast_config": null,
    "spectra_config": null,
    "mempool_config": null,
    "catchup_config": null,
    "solo_config": {
        "leader": {
            "public_key": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
            "address": ""
        },
        "quorum": [
            {
                "public_key": "mClRx95/kb3hdwE0tvMBt+S4pPTOJQoc5fn9fs3lK9c=",
                "address": "localhost:50000"
            },
            {
                "public_key": "JEVKqEYDTT2LuREHl8lxMstj+ReU21SNmm/wMknuoS4=",
                "address": "localhost:50001"
            },
            {
                "public_key": "eJOCvsH4nfQSZdKTySLutdWojYpGHAbs8H5iAKzMXQA=",
                "address": "localhost:50002"
            }
        ]
    },
    "baton_config": null
}
EOF
cat > config-quorum-0.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "mClRx95/kb3hdwE0tvMBt+S4pPTOJQoc5fn9fs3lK9c=",
                "address": "localhost:50000"
            },
            "secret_key": "q7XHGyKWwdDkklzPIQvNsgZl3qXgKi6rOlk+/aog0CGYKVHH3n+RveF3ATS28wG35Lik9M4lChzl+f1+zeUr1w=="
        },
        "quorum_set": {
            "threshold": 2,
            "validators": [
                "cntDcxDPbihsmwdDfva5rZruKFskTcyH7W5f4mzH0vk=",
                "VTMeWIjYY+4wIYTH4puIbFrEZ+Gb1V3c/XaQpYrrPg8=",
                "5CKAwEY+HZuSoA8fn3Me/NfX/s44r8CT/QXJJ8yn4qs="
            ],
            "inner_sets": [
            ]
        },
        "known_validators": [
            {
                "public_key": "mClRx95/kb3hdwE0tvMBt+S4pPTOJQoc5fn9fs3lK9c=",
                "address": "localhost:50000"
            },
            {
                "public_key": "JEVKqEYDTT2LuREHl8lxMstj+ReU21SNmm/wMknuoS4=",
                "address": "localhost:50001"
            },
            {
                "public_key": "eJOCvsH4nfQSZdKTySLutdWojYpGHAbs8H5iAKzMXQA=",
                "address": "localhost:50002"
            }
        ]
    },
    "wallet_config": {
        "secret_key": "AQID",
        "users": [
        ]
    },
    "discovery_config": {
        "advertise_interval_msec": "5000",
        "advertise_par": 5,
        "snapshot_ttl_msec": "7000"
    },
    "router_config": {
        "tree_fan_out": 5
    },
    "broadcast_config": {
        "multicast_tree_ttl_msec": "1000",
        "chunk_size": "1000000"
    },
    "spectra_config": {
        "num_payload_channels": 5,
        "num_parity_channels": 2,
        "partial_ttl_msec": "1000"
    },
    "mempool_config": {
        "proposed_batch_ttl_msec": "5000",
        "transaction_ttl_msec": "3000",
        "submission_lot_size": "1000",
        "submission_lot_wait_msec": "50"
    },
    "catchup_config": {
        "catchup_block_ttl_sec": "5",
        "catchup_block_age": "5"
    },
    "solo_config": {
        "leader": {
            "public_key": "lwyboIFFL6fwlLu6UZZvSmTJEI6m/sbR3LutOOJeD1Y=",
            "address": "localhost:60000"
        },
        "quorum": [
        ]
    },
    "baton_config": {
        "heartbeat_msec": "100"
    }
}
EOF
cat > config-quorum-1.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "JEVKqEYDTT2LuREHl8lxMstj+ReU21SNmm/wMknuoS4=",
                "address": "localhost:50001"
            },
            "secret_key": "pOv5G/Wl5i4po8jI3PZImgnkVGitXBpB7is7wF9D6fQkRUqoRgNNPYu5EQeXyXEyy2P5F5TbVI2ab/AySe6hLg=="
        },
        "quorum_set": {
            "threshold": 2,
            "validators": [
                "cntDcxDPbihsmwdDfva5rZruKFskTcyH7W5f4mzH0vk=",
                "VTMeWIjYY+4wIYTH4puIbFrEZ+Gb1V3c/XaQpYrrPg8=",
                "5CKAwEY+HZuSoA8fn3Me/NfX/s44r8CT/QXJJ8yn4qs="
            ],
            "inner_sets": [
            ]
        },
        "known_validators": [
            {
                "public_key": "mClRx95/kb3hdwE0tvMBt+S4pPTOJQoc5fn9fs3lK9c=",
                "address": "localhost:50000"
            },
            {
                "public_key": "JEVKqEYDTT2LuREHl8lxMstj+ReU21SNmm/wMknuoS4=",
                "address": "localhost:50001"
            },
            {
                "public_key": "eJOCvsH4nfQSZdKTySLutdWojYpGHAbs8H5iAKzMXQA=",
                "address": "localhost:50002"
            }
        ]
    },
    "wallet_config": {
        "secret_key": "AQID",
        "users": [
        ]
    },
    "discovery_config": {
        "advertise_interval_msec": "5000",
        "advertise_par": 5,
        "snapshot_ttl_msec": "7000"
    },
    "router_config": {
        "tree_fan_out": 5
    },
    "broadcast_config": {
        "multicast_tree_ttl_msec": "1000",
        "chunk_size": "1000000"
    },
    "spectra_config": {
        "num_payload_channels": 5,
        "num_parity_channels": 2,
        "partial_ttl_msec": "1000"
    },
    "mempool_config": {
        "proposed_batch_ttl_msec": "5000",
        "transaction_ttl_msec": "3000",
        "submission_lot_size": "1000",
        "submission_lot_wait_msec": "50"
    },
    "catchup_config": {
        "catchup_block_ttl_sec": "5",
        "catchup_block_age": "5"
    },
    "solo_config": {
        "leader": {
            "public_key": "lwyboIFFL6fwlLu6UZZvSmTJEI6m/sbR3LutOOJeD1Y=",
            "address": "localhost:60000"
        },
        "quorum": [
        ]
    },
    "baton_config": {
        "heartbeat_msec": "100"
    }
}
EOF
cat > config-quorum-2.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "eJOCvsH4nfQSZdKTySLutdWojYpGHAbs8H5iAKzMXQA=",
                "address": "localhost:50002"
            },
            "secret_key": "Nd/8nii5K7jrvNChYwF8R0L7U2PxFsQbURvsR/AtkWx4k4K+wfid9BJl0pPJIu611aiNikYcBuzwfmIArMxdAA=="
        },
        "quorum_set": {
            "threshold": 2,
            "validators": [
                "cntDcxDPbihsmwdDfva5rZruKFskTcyH7W5f4mzH0vk=",
                "VTMeWIjYY+4wIYTH4puIbFrEZ+Gb1V3c/XaQpYrrPg8=",
                "5CKAwEY+HZuSoA8fn3Me/NfX/s44r8CT/QXJJ8yn4qs="
            ],
            "inner_sets": [
            ]
        },
        "known_validators": [
            {
                "public_key": "mClRx95/kb3hdwE0tvMBt+S4pPTOJQoc5fn9fs3lK9c=",
                "address": "localhost:50000"
            },
            {
                "public_key": "JEVKqEYDTT2LuREHl8lxMstj+ReU21SNmm/wMknuoS4=",
                "address": "localhost:50001"
            },
            {
                "public_key": "eJOCvsH4nfQSZdKTySLutdWojYpGHAbs8H5iAKzMXQA=",
                "address": "localhost:50002"
            }
        ]
    },
    "wallet_config": {
        "secret_key": "AQID",
        "users": [
        ]
    },
    "discovery_config": {
        "advertise_interval_msec": "5000",
        "advertise_par": 5,
        "snapshot_ttl_msec": "7000"
    },
    "router_config": {
        "tree_fan_out": 5
    },
    "broadcast_config": {
        "multicast_tree_ttl_msec": "1000",
        "chunk_size": "1000000"
    },
    "spectra_config": {
        "num_payload_channels": 5,
        "num_parity_channels": 2,
        "partial_ttl_msec": "1000"
    },
    "mempool_config": {
        "proposed_batch_ttl_msec": "5000",
        "transaction_ttl_msec": "3000",
        "submission_lot_size": "1000",
        "submission_lot_wait_msec": "50"
    },
    "catchup_config": {
        "catchup_block_ttl_sec": "5",
        "catchup_block_age": "5"
    },
    "solo_config": {
        "leader": {
            "public_key": "lwyboIFFL6fwlLu6UZZvSmTJEI6m/sbR3LutOOJeD1Y=",
            "address": "localhost:60000"
        },
        "quorum": [
        ]
    },
    "baton_config": {
        "heartbeat_msec": "100"
    }
}
EOF
nohup chuckd --config config-leader.json --instance leader --seed 8502489525653699386 --metrics :2222 solo leader --addr :60000 --delay 100 > leader.log 2>&1 &
nohup chuckd --config config-quorum-0.json --instance quorum-0 --seed 6311280341148717924 --metrics :2223 bench testcluster --tps 1000 --out-addr :50000 --w-addr :40000 > quorum-0.log 2>&1 &
nohup chuckd --config config-quorum-1.json --instance quorum-1 --seed 2622264106489803354 --metrics :2224 bench testcluster --tps 1000 --out-addr :50001 --w-addr :40001 > quorum-1.log 2>&1 &
nohup chuckd --config config-quorum-2.json --instance quorum-2 --seed 7964169681022589069 --metrics :2225 bench testcluster --tps 1000 --out-addr :50002 --w-addr :40002 > quorum-2.log 2>&1 &

