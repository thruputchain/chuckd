INTEGRATION WITH SCP

- [x] why does the current baton impl allow re-nomination for the same slot?
  - because baton currently begins re-nomination after some TTL (in canNominate)

- [ ] handle missing txs
  - [x] add coordinator rpc for retrieving missing txs
  - [x] hold txs in mempool while txs not committed
  - [ ] on scp validation, attempt to retrieve missing txs until success or newer externalization
    - [ ] write proposer to nominations!
    - [ ] fill in repair config
  - [ ] update baton to disallow nominations if there is an existing one
    - [ ] get rid of the nomination ttl
    - [ ] update scp mocks and unit tests


- [ ] SCP/baton
  - [ ] cancel all timers below an externalized slot
  - [ ] call stop_nomination on externalizations, etc??
- [ ] who takes care of purging live SCP slots below a newly externalized slot?
    - [ ] add purge_slots to SCP_CONTROL interface



ROADMAP

- [ ] intro BlockHeader { height, merkle_hash, later on also txset hash }
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
