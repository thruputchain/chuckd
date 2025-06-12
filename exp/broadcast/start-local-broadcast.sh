
#!/bin/sh
set -xe

cat > config-0.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "5pPqDw5hWN89ffL1XlYx7Ds7KDRSi3KUanU3L8ne/O0=",
                "address": "localhost:4000"
            },
            "secret_key": "+NvTnCxRuJI/YGFFxWC9XTog3t20/rhhghU+ccUT4qHmk+oPDmFY3z198vVeVjHsOzsoNFKLcpRqdTcvyd787Q=="
        },
        "quorum_set": {
            "threshold": 2,
            "validators": [
                "XL/OTyA1xjejFNT+2ixnfEEuqxTLPzyroAvbG7gaAUc=",
                "+cI9b3TDV6oy2myqOHat1V6QuRuP8KdzoOdub7MSaVM=",
                "fT7ULmdL9z13ifOQCX80sYL1XI04D3iaYn6nGy3UULo="
            ],
            "inner_sets": [
            ]
        },
        "known_validators": [
            {
                "public_key": "5pPqDw5hWN89ffL1XlYx7Ds7KDRSi3KUanU3L8ne/O0=",
                "address": "localhost:4000"
            },
            {
                "public_key": "Cgu5yfnvokD+4sJddjHjaRPolR/QOFD+jw2hzOVzb5g=",
                "address": "localhost:4001"
            },
            {
                "public_key": "mwVXvJ9fMaSbvyCAUskk+RJLjEDC2nXR48dU5baw0LQ=",
                "address": "localhost:4002"
            }
        ]
    },
    "wallet_config": null,
    "discovery_config": null,
    "router_config": null,
    "broadcast_config": {
        "multicast_tree_ttl_msec": "3000",
        "chunk_size": "64000"
    },
    "spectra_config": null,
    "mempool_config": null,
    "catchup_config": null,
    "solo_config": null,
    "baton_config": null
}
EOF

chuckd -v -l info --config config.json-0 \
	--instance node-0 \
	--seed 5707798609610607655 \
	--metrics :2222 \
	bench broadcast \
	--addr :4000 \
	 \
	--fan-out 5 \
	--chunk-per-chan 100 \
	--bytes-per-sec 128000 \
	--num-iterations 1 \
	> bb-0.log 2>&1 &


#!/bin/sh
set -xe

cat > config-1.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "Cgu5yfnvokD+4sJddjHjaRPolR/QOFD+jw2hzOVzb5g=",
                "address": "localhost:4001"
            },
            "secret_key": "YjSDTE8VWcuUw2J085/dMGsn4Ek8iDUFLxyIX58KTZ4KC7nJ+e+iQP7iwl12MeNpE+iVH9A4UP6PDaHM5XNvmA=="
        },
        "quorum_set": {
            "threshold": 2,
            "validators": [
                "XL/OTyA1xjejFNT+2ixnfEEuqxTLPzyroAvbG7gaAUc=",
                "+cI9b3TDV6oy2myqOHat1V6QuRuP8KdzoOdub7MSaVM=",
                "fT7ULmdL9z13ifOQCX80sYL1XI04D3iaYn6nGy3UULo="
            ],
            "inner_sets": [
            ]
        },
        "known_validators": [
            {
                "public_key": "5pPqDw5hWN89ffL1XlYx7Ds7KDRSi3KUanU3L8ne/O0=",
                "address": "localhost:4000"
            },
            {
                "public_key": "Cgu5yfnvokD+4sJddjHjaRPolR/QOFD+jw2hzOVzb5g=",
                "address": "localhost:4001"
            },
            {
                "public_key": "mwVXvJ9fMaSbvyCAUskk+RJLjEDC2nXR48dU5baw0LQ=",
                "address": "localhost:4002"
            }
        ]
    },
    "wallet_config": null,
    "discovery_config": null,
    "router_config": null,
    "broadcast_config": {
        "multicast_tree_ttl_msec": "3000",
        "chunk_size": "64000"
    },
    "spectra_config": null,
    "mempool_config": null,
    "catchup_config": null,
    "solo_config": null,
    "baton_config": null
}
EOF

chuckd -v -l info --config config.json-1 \
	--instance node-1 \
	--seed 3886150534851568708 \
	--metrics :2223 \
	bench broadcast \
	--addr :4001 \
	 \
	--fan-out 5 \
	--chunk-per-chan 100 \
	--bytes-per-sec 128000 \
	--num-iterations 1 \
	> bb-1.log 2>&1 &


#!/bin/sh
set -xe

cat > config-2.json << EOF

{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "mwVXvJ9fMaSbvyCAUskk+RJLjEDC2nXR48dU5baw0LQ=",
                "address": "localhost:4002"
            },
            "secret_key": "DrH7paWrbMWhtVYKAS0LoA2QKL4OYeyJbJjSkFPf/JubBVe8n18xpJu/IIBSyST5EkuMQMLaddHjx1TltrDQtA=="
        },
        "quorum_set": {
            "threshold": 2,
            "validators": [
                "XL/OTyA1xjejFNT+2ixnfEEuqxTLPzyroAvbG7gaAUc=",
                "+cI9b3TDV6oy2myqOHat1V6QuRuP8KdzoOdub7MSaVM=",
                "fT7ULmdL9z13ifOQCX80sYL1XI04D3iaYn6nGy3UULo="
            ],
            "inner_sets": [
            ]
        },
        "known_validators": [
            {
                "public_key": "5pPqDw5hWN89ffL1XlYx7Ds7KDRSi3KUanU3L8ne/O0=",
                "address": "localhost:4000"
            },
            {
                "public_key": "Cgu5yfnvokD+4sJddjHjaRPolR/QOFD+jw2hzOVzb5g=",
                "address": "localhost:4001"
            },
            {
                "public_key": "mwVXvJ9fMaSbvyCAUskk+RJLjEDC2nXR48dU5baw0LQ=",
                "address": "localhost:4002"
            }
        ]
    },
    "wallet_config": null,
    "discovery_config": null,
    "router_config": null,
    "broadcast_config": {
        "multicast_tree_ttl_msec": "3000",
        "chunk_size": "64000"
    },
    "spectra_config": null,
    "mempool_config": null,
    "catchup_config": null,
    "solo_config": null,
    "baton_config": null
}
EOF

chuckd -v -l info --config config.json-2 \
	--instance node-2 \
	--seed 177360397769365151 \
	--metrics :2224 \
	bench broadcast \
	--addr :4002 \
	 \
	--fan-out 5 \
	--chunk-per-chan 100 \
	--bytes-per-sec 128000 \
	--num-iterations 1 \
	> bb-2.log 2>&1 &

