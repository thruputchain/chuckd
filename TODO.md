SCHEDULER INTEGRATION

- [x] handle missing txs
- [x] make number of blocks, beyond which catchup doesn't help, dynamically configurable
- [ ] emitted events need not include large ValidatableBlock object

- [ ] fix events.go
- [ ] fix tests in baton

ROADMAP

- [ ] intro BlockHeader { timestamp, height, merkle_hash, txset hash, updates? }
- [ ] design scp values 
  - [ ] separate block txset distribution from scp value
  - [ ] include timestamp and config updates


- [ ] pebble db in catchup cache
- [ ] connection discipline and timeouts
  - [ ] timeouts on all rpc calls
- [ ] tier-1 router with weighted sampling


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
