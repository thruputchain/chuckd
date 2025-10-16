CONSENSUS VALUE

- [ ] validate timestamp
  - not in the future
  - monotonically increasing
- [ ] verify that catchup blocks (proposal+upgrades) include valid upgrades
  - current block hash linking algorithm does not consider updates

- [ ] catchup blocks must include consensus value (timestamp and upgrades)

- [ ] purpose of extract valid value?
- [ ] purpose of MaybeValid?

ROADMAP

- [ ] verify that the hashes in nomination and catchup blocks match the contents of the block
- [ ] separate block txset distribution from scp value

- [ ] upgrade mechanism
  - [ ] baton pulls pending upgrades from persistence and places them in consensus values
  - [ ] extract valid value strips unrecognizable upgrades?
  - [ ] NOTE that baton.validated caches block validations, but on the upgrade validations

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
