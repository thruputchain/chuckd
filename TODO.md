SCHEDULER INTEGRATION

- [ ] reject validations below or at db tip
- [ ] wait when asked to validate ahead of dbtip+1

ROADMAP

- [ ] intro BlockHeader { timestamp, height, merkle_hash, txset hash, updates? }
  - [ ] verify that the hashes in nomination and catchup blocks match the contents of the block

- [ ] design scp values 
  - [ ] separate block txset distribution from scp value
  - [ ] include timestamp and config updates


- [ ] connection discipline and timeouts
  - [ ] timeouts on all rpc calls


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
