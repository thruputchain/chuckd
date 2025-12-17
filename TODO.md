SCHEDULER INTEGRATION

- [ ] return immediately from submit_vetted
  - increase submission_lot_size to 50'000
  - tune number of workers in multiplex
  - take concurrent connections?
  - rpc parallel?
  - transactions as byte blob

BUGS

- [x] `repaired consensus value hash mismatch` https://github.com/groundhogchain/chuckd/actions/runs/20147245728/job/57830875188
  - [ ] what if consensus value for a non-externalized block is missing and externalizations have gone beyond?
- [ ] validators must be able to respond to missing txs request even for past blocks

HOGTOOLS

- [ ] use a single build tree in experiments
- [ ] extract cause of failure from logs?

- [ ] Here's the exact list of things I want from a failed program in a CI run:
  - [ ] Git commit hash
  - [ ] Path of source tree when everything was compiled
  - [ ] Exact cmake command line that was used to configure the build directory
  - [ ] Exact command line that was used to execute the failing program
  - [ ] Standard error of failing program
  - [ ] Binary of failing program
  - [ ] Core dump of failed run

- [ ] dump stack trace when scpd killed

QUESTIONS

- [ ] why `dogwood dropping partial incomplete transmission` appears before `counterparties online`?







SCP MISC

- [ ] purpose of MaybeValid?
- [ ] ask stellar folks how they deal with messages coming after an externalization

UPGRADES

- [ ] verify that catchup blocks (proposal+upgrades) include valid upgrades
  - [ ] current block hash linking algorithm does not consider upgrades

- [ ] baton pulls pending upgrades from persistence and places them in consensus values
- [ ] extract valid value strips unrecognizable upgrades?
- [ ] NOTE that baton.validated caches block validations, but on the upgrade validations
- [ ] interaction between storage and protocol upgrades???

CATCHUP BLOCK PERSISTENCE

LIVE UPDATE QUORUM SET

RESEARCH

- [ ]  https://commonware.xyz/blogs/zoda

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
