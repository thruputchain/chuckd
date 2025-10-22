CONSENSUS VALUE

- [x] add storage to system
  - [x] persist scp envelopes
- [x] add storage access to baton handler
  - [x] persist consensus values
  - [x] track consensus value corresponding to db tip
    - [x] on startup, load tip cv unless height == 0
    - [x] on startup, load cvs above tip
    - [x] update tip cv on commit

- [x] validate timestamp
  - [x] not in the future
  - [x] monotonically increasing

- [x] add hash of prev consensus value on nomination
- [x] check hash of prev consensus value on validation
- [x] check catchup block validity
  - [x] included txs match txs hashes in nomination
- [ ] check correctness when chaining/linking catchup blocks in compilation

===========================

- [ ] verify that catchup blocks (proposal+upgrades) include valid upgrades
  - current block hash linking algorithm does not consider upgrades
  - consensus value must include the hash of the previous consensus value to enable correct chaining here

- [ ] purpose of extract valid value?
- [ ] purpose of MaybeValid?

ROADMAP

- [ ] verify that the hashes in nomination and catchup blocks match the contents of the block
- [ ] separate block txset distribution from scp value

- [ ] upgrade mechanism
  - [ ] baton pulls pending upgrades from persistence and places them in consensus values
  - [ ] extract valid value strips unrecognizable upgrades?
  - [ ] NOTE that baton.validated caches block validations, but on the upgrade validations
  - [ ] interaction between storage and protocol upgrades???

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
