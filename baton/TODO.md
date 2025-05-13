- [ ] should catchup blocks be requested for externalizations with missing batches?
  - can externalization arrive before batches in normal flow?

- [ ] test
  - [ ] can catchup requests spin too fast if not all blocks are returned?
  - [x] txs cannot be submitted if older than txTTL; enforced by expiration in txpool and batchpool
  - [ ] capture disseminations
  - [ ] ensure committed transactions are marked regardless of the commit path (catchup, cached, etc)


- [ ] does a proposal need to be signed by the source nominator?



- [ ] implement SCP controller 
- [ ] implement catchup service!
- [ ] implement live handler
- [ ] plumb mempools in system
- [ ] plumb baton in system
