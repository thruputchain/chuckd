- [ ] nominate signal without transactions to nominate?
  - creates empty block in db
  - sends empty batch
- [x] txs cannot be submitted if older than txTTL
  - enforced by expiration in txpool and batchpool
- [*] ensure committed transactions are marked regardless of the commit path (catchup, cached, etc)


- [ ] test
  - [ ] capture disseminations

- [ ] does a proposal need to be signed by the source nominator?



- [ ] implement SCP controller 
- [ ] implement catchup service!
- [ ] implement live handler
- [ ] plumb mempools in system
- [ ] plumb baton in system
