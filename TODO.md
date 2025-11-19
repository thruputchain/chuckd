MISC

- [ ] purpose of MaybeValid?
- [ ] ask stellar folks how they deal with messages coming after an externalization

HOGSCHED

- [ ] pass branch or commit to hog schedules
- [ ] .gitconfig conflicts
- [ ] after-script (gets payload script exit code)

OPEN

- [ ] why `dogwood dropping partial incomplete transmission` appears before `counterparties online`

UPGRADES

- [ ] verify that catchup blocks (proposal+upgrades) include valid upgrades
  - [ ] current block hash linking algorithm does not consider upgrades

- [ ] baton pulls pending upgrades from persistence and places them in consensus values
- [ ] extract valid value strips unrecognizable upgrades?
- [ ] NOTE that baton.validated caches block validations, but on the upgrade validations
- [ ] interaction between storage and protocol upgrades???

CATCHUP BLOCK PERSISTENCE

UPDATE QUORUM SET

BACKLOG

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
