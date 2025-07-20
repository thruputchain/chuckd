- [x] fix rpc driver race 244/265

RPC OPTIMIZATIONS
    - [x] batch propose/validate (3x)
    - [x] driver.Go no goroutine (0.5x)
    - [x] pause GC (2x)
    - [x] use a sync.Pool for rpc.Messages in rpc.Transport
    - [?] xdr arrays with pointer elements
    - [/] buffer pool for xdrReadN (not showing gains in experiment)

- [ ] why hog has lower performance than mac?
  - [ ] context switches? hog_cs = 21K
  - [ ] cache misses? hog_miss = 13%


ROADMAP

- [ ] sign over payload to bind source and payload (prevent replay)

- [ ] pebble db in catchup cache
- [x] quic support (includes cached connections)
- [ ] connection discipline and timeouts
  - [ ] timeouts on all rpc calls
- [ ] tier-1 router with weighted sampling
- [ ] scp implementation
  - [ ] reuse multicast trees for efficient broadcast of small (scp) messages
- [ ] consider using unreliable broadcast (quic datagram) and still error correct across trees/shreds


XDR
- [ ] equality for xdr types
- [ ] hash for xdr types
- [ ] implement SprintXDR (use xdr textual)


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
