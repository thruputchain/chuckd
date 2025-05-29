- [x] fix rpc driver race 244/265


- [ ] reuse broadcast trees (because each tree used for one bag ~200KB)




- [ ] system
  - [ ] live SubmitTx
  - [ ] wallet use SubmitTx
- [ ] baton
  - [ ] handler
  - [ ] logic
  - [ ] baton_test
- [ ] address XXX
- [ ] cleanup batch/proposal structures





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
