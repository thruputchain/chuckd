# Dogwood: Dissemination system for Thruput

Dogwood is the code name for Thruput's dissemination subsystem. It can be
deployed as a standalone system, which performs broadcasts on behalf of a
local client. The local client controls the Dogwood node via the Dogwood
XDR/RPC API (`dogwood_api.x` in the
[Thruput protocol](https://github.com/thruputchain/thruput-protocol)).

# Deployment

To deploy a Dogwood node:

1. Build `chuckd` from this repository (`make coordinator` leaves it at
   `./chuckd.dev`), or install it:

   ```sh
   go install github.com/thruputchain/chuckd/chuckd@latest
   ```

2. Generate this node's identity. The command writes a configuration with a
   fresh key pair, readable by the current user only:

   ```sh
   chuckd config init --config config.json
   ```

   Then fill in the rest. The example below is what a complete
   configuration looks like for a three-node network; the keys in it are
   throwaway keys generated for this page, so replace all of them with your
   own. Public keys are base58 Ed25519 keys and secret keys are hex.
   Durations are strings, in milliseconds.

   ```json
   {
     "validator_config": {
       "private_info": {
         "public_info": {
           "public_key": {"Ed25519": {"ed25519": "95xSdV3FtsZZWFexcK2bfBAYVF19Jx7FqWeeXzpsg3ed"}},
           "address": "node1.example.net:7177"
         },
         "secret_key": {"Ed25519": {"ed25519": "2aeb184dca52123426114a5e26bb34525387be1caabd6d0845900345979792ac782281c97780c0a0f8e8ca04b7f1a02921d18d636101f12f5ab4deb6def63ea6"}}
       },
       "known_validators": [
         {
           "public_key": {"Ed25519": {"ed25519": "95xSdV3FtsZZWFexcK2bfBAYVF19Jx7FqWeeXzpsg3ed"}},
           "address": "node1.example.net:7177"
         },
         {
           "public_key": {"Ed25519": {"ed25519": "4MRkH62yJ9XQfgEMHi4ncoLTdk4frg4fiCyzKwpD1P2A"}},
           "address": "node2.example.net:7177"
         },
         {
           "public_key": {"Ed25519": {"ed25519": "DuPkzZv7aHvEAutfHno93H2ZswmkYHPfcRcbHf1UFi2q"}},
           "address": "node3.example.net:7177"
         }
       ]
     },
     "router_config": {
       "tree_fan_out": 5
     },
     "dogwood_config": {
       "route_count": 20,
       "route_time_to_live_msec": "300000",
       "route_max_latency_msec": "2000",
       "endpoint_max_pending": 4000,
       "endpoint_num_clients": 10,
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
     "discovery_config": null,
     "scp_config": null,
     "mempool_config": null,
     "catchup_config": null,
     "baton_config": null,
     "repair_config": null,
     "storage_config": null
   }
   ```

   The settings, in order:

   - `known_validators`: every node in the network that this node knows of.
   - `router_config.tree_fan_out`: maximum fan out of the multicast trees
     used for dissemination.
   - `route_count`: number of routes (multicast trees) to generate for
     broadcasts.
   - `route_time_to_live_msec`: recycle multicast routes this often.
   - `route_max_latency_msec`: assume a dissemination reaches all nodes
     within this time.
   - `endpoint_max_pending`: maximum pending requests per endpoint.
   - `endpoint_num_clients`: TCP connections per endpoint.
   - `endpoint_idle_ttl_msec`: disconnect from an endpoint idle for this
     long.
   - `endpoint_dial_timeout_msec`: how long to wait to establish a
     connection.
   - `endpoint_redial_delay_msec`: redial an unresponsive endpoint after this
     delay.
   - `endpoint_call_timeout_msec`: how long to wait for replies to RPC calls.
   - `transmission_partial_max_ttl_msec`: retain partial transmissions for at
     most this long.
   - `transmission_partial_idle_ttl_msec`: drop a partial transmission if no
     new packets arrive within this time.

3. Start the Dogwood server:

   ```sh
   chuckd -l info --config config.json --stack --seed 12345678 --metrics :2222 \
       sys dogwood --in-addr localhost:7777 --out-addr :7177 \
       2> dogwood.log 1> dogwood.out
   ```

   - `-l`: log level, one of `debug`, `info`, `error` (use `error` in
     production).
   - `--stack`: print a stack trace when the server is killed or interrupted.
   - `--seed`: seeds the randomness used by the server.
   - `--metrics`: the OpenTelemetry/Prometheus metrics endpoint.
   - `--in-addr`: the server address for the local Dogwood API.
   - `--out-addr`: the public address for communicating with peers; its port
     is the one in this node's `address` above.
