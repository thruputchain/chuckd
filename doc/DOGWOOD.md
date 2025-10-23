# Dogwood: Dissemination system for Groundhog

Dogwood is the code name for Groundhog's dissemination subsystem.
It can be deployed as a standalone system, which performs broadcasts on behalf of a local client.
The local client controls the Dogwood node via the [Dogwood XDR/RPC API](https://github.com/groundhogchain/protocol/blob/main/dogwood_api.x).

# Deployment

To deploy a Dogwood node:

1. Set up your Go and Git environment to [work with private Go repositories](https://fueled.com/blog/using-private-modules-in-go/).

2. Install the Dogwood software:

```sh
go install github.com/groundhogchain/chuckd@dogwood-v1.0.0
```

3. Prepare a JSON config file `config.json` following the [XDR configuration protocol](https://github.com/groundhogchain/protocol/blob/main/dogwood_api.x#L5), e.g.

```json
{
    "validator_config": {
        "private_info": {
            "public_info": {
                "public_key": "3FK04muEJGo2jCm/reF/KlqTvEFzDKUzoFzS8FaNYpg=", # public keys are Base64 32-byte Ed25519 keys
                "address": "hog5:7177"
            },
            "secret_key": "CT+aTotBXKAQIt+yIiUKEMSDwejYPGdCdXrMi07hbjTcUrTia4QkajaMKb+t4X8qWpO8QXMMpTOgXNLwVo1imA=="
        },
        "quorum_set": { # quorum_set can be left empty
            "threshold": 0,
            "validators": [],
            "inner_sets": []
        },
        "known_validators": [ # known_validators lists all nodes in the network, known to this node
            {
                "public_key": "3FK04muEJGo2jCm/reF/KlqTvEFzDKUzoFzS8FaNYpg=",
                "address": "hog5:7177"
            },
            {
                "public_key": "PxR+5bWjVdjdov9nxKnvlrO8Rtdawhq/wtzENCESuro=",
                "address": "hog6:7177"
            },
            {
                "public_key": "SHE76vO9juFofGPNh3yi4WnLc2Y673JkaJUv9xUIxqw=",
                "address": "hog7:7177"
            }
        ]
    },
    "router_config": {
        "tree_fan_out": 5 # maximum fan out of the multicast trees used for dissemination
    },
    "dogwood_config": {
        "route_count": 20, # number of routes (multicast trees) to generate for broadcasts
        "route_time_to_live_msec": "300000", # recycle multicast routes every 5 minutes
        "route_max_latency_msec": "2000", # assume dissemination propagates all nodes within 2 secs
        "endpoint_max_pending": 4000, # maximum pending requests per endpoint
        "endpoint_num_clients": 10, # number of TCP connections per endpoint
        "endpoint_idle_ttl_msec": "60000", # disconnect from an endpoint if idle for 1 minute
        "endpoint_dial_timeout_msec": "2000", # wait at most 1 sec to establish a connection
        "endpoint_redial_delay_msec": "100", # redial unresponsive endpoints after 100 msec
        "endpoint_call_timeout_msec": "2000", # wait at most 2 sec for replies to rpc calls
        "stream_receiver_worker_count": 512,
        "stream_sender_worker_count": 32,
        "transmission_receiver_decode_worker_count": 512,
        "transmission_sender_worker_count": 32,
        "transmission_partial_max_ttl_msec": "10000", # retain partial transmissions for at most 10 seconds
        "transmission_partial_idle_ttl_msec": "1000" # drop partial transmissions if no new packets arrive withing 1 sec
    },
}
```

4. Deploy the Dogwood server:

```sh
chuckd \
    -l info \ # log levels: "debug", "info", "error" (use "error" for production)
    --config config.json \
	--stack \ # --stack will cause the server to print a stack trace when killed or interrupted
	--seed 12345678 \ # --seed seeds the randomness used by the server
	--metrics :2222 \ # --metrics specified the OpenTelemetry/Prometheus metrics endpoint
	sys dogwood \
	--in-addr localhost:7777 \ # --in-addr is the server address for the local Dogwood API
    --out-addr :9999 \ # --out-addr is the public address for communicating with peers
    2> dogwood.log 1> dogwood.out

```
