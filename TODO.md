CATCHUP BLOCK PERSISTENCE

- [ ] loadgen test catchup (3-node, 1 node restarts)
  - [ ] Why same cv validated multiple times?
    - [ ] Are multiple validation requests initiating validation before previous finished and cached?

SCP

- [ ] are our scp envelopes persisted?

QUESTIONS

- [ ] why `dogwood dropping partial incomplete transmission` appears before `counterparties online`?

OPTIMIZATIONS

- [ ] ask scp whether we should bother nominating this round

UPGRADES

- [ ] verify that catchup blocks (proposal+upgrades) include valid upgrades
  - [ ] current block hash linking algorithm does not consider upgrades

- [ ] baton pulls pending upgrades from persistence and places them in consensus values
- [ ] extract valid value strips unrecognizable upgrades?
- [ ] NOTE that baton.validated caches block validations, but on the upgrade validations
- [ ] interaction between storage and protocol upgrades???

LIVE UPDATE QUORUM SET

RESEARCH

- [ ] https://commonware.xyz/blogs/zoda

BACKLOG

- [ ] connection discipline and timeouts
  - [ ] timeouts on all rpc calls

IDEAS

- Marco: nodes disseminate network topology (instead of trees), can even use set reconciliation
