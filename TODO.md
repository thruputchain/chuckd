- [x] fix rpc driver race 244/265

- [x] upgrade for geoff's new executor error codes

- [ ] try conditional variable / mutex instead of the per-call channel in driver.go
- [ ] add batching calls to exec interface

- [?] Bcs look for calldata arguments

- [ ] reuse broadcast trees (because each tree used for one bag ~200KB)





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
- [ ] setup us/asia benchmark on aws kubernetes
- [ ] setup prometheus on aws


GITHUB
- [ ] code coverage
- [ ] code builds and tests
