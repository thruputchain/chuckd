
FINAL INTEGRATION

- [ ] deploy SCP sidecar (chicken and egg connection)

ROADMAP

- [ ] intro BlockHeader { height, merkle_hash, later on also txset hash }



- [ ] pebble db in catchup cache
- [ ] connection discipline and timeouts
  - [ ] timeouts on all rpc calls
- [ ] tier-1 router with weighted sampling
- [ ] scp implementation
  - [ ] reuse multicast trees for efficient broadcast of small (scp) messages
- [ ] consider using unreliable broadcast (quic datagram) and still error correct across trees/shreds


XDR
- [ ] equality for xdr types
- [ ] hash for xdr types
- [ ] implement SprintXDR (use xdr txrep)


INFRA & BENCHMARK
- [ ] setup us/asia benchmark on aws kubernetes
- [ ] setup prometheus on aws


GITHUB
- [ ] code coverage
- [ ] code builds and tests



IDEAS

- Marco: nodes disseminate network topology (instead of trees), can even use set reconciliation
- active queue management https://en.wikipedia.org/wiki/CoDel

- multiple channels per worker pools
- or, one goroutine that stays with a packet through all its life
    https://vimeo.com/649009599

- raptorQ
  - https://datatracker.ietf.org/doc/html/rfc6330
  - https://datatracker.ietf.org/ipr/2554/
