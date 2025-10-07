INTEGRATION WITH SCP

- [x] why does the current baton impl allow re-nomination for the same slot?
  - because baton currently begins re-nomination after some TTL (in canNominate)

- [x] handle missing txs
  - [x] add coordinator rpc for retrieving missing txs
  - [x] hold txs in mempool while txs not committed
  - [x] on scp validation, attempt to retrieve missing txs until success or newer externalization
    - [x] write proposer to nominations!
    - [x] fill in repair config
  - [x] use cache for validations
  - [x] update baton to disallow nominations if there is an existing one
    - [x] get rid of the nomination ttl
    - [x] update scp mocks and unit tests


- [x] SCP/baton
  - [x] stop all timers below an externalized slot (scp_server::control_kernel)
  - [x] call stop_nomination on externalizations
  - [x] value_externalized can be called multiple times for the same slot
- [x] who takes care of purging live SCP slots below a newly externalized slot? (scp_server::control_kernel)



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
