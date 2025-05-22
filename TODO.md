- [ ] Validatevalue should block until the batches arrive? 


ROADMAP

- [ ] sign over payload to bind source and payload (prevent replay)

- [ ] pebble db in catchup cache
- [ ] quic support (includes cached connections)
- [ ] connection discipline and timeouts
  - [ ] timeouts on all rpc calls
- [ ] tier-1 router with weighted sampling
- [ ] scp implementation
  - [ ] reuse multicast trees for efficient broadcast of small (scp) messages
- [ ] consider using unreliable broadcast (quic datagram) and still error correct across trees/shreds


XDR
- [ ] implement SprintXDR (use xdr textual)

INFRA & BENCHMARK
- [x] setup benchmark on local docker (three nodes)
- [ ] setup us/asia benchmark on aws kubernetes
- [ ] setup prometheus on aws


GITHUB
- [ ] code coverage
- [ ] code builds and tests
