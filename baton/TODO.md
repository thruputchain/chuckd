- [x] txs cannot be submitted if older than txTTL
  - enforced by expiration in txpool and batchpool

- [*] ensure committed transactions are marked regardless of the commit path (catchup, cached, etc)
  - [ ] assert in tests


- [ ] test
  - [ ] capture disseminations


- [ ] does a proposal need to be signed by the source nominator?



- [ ] implement SCP controller 
- [ ] implement catchup service!
- [ ] implement live handler
- [ ] plumb mempools in system
- [ ] plumb baton in system
