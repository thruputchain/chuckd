CATCHUP BLOCK PERSISTENCE

- [ ] loadgen test catchup (3-node, 1 node restarts)

SCP

- [x] are our scp envelopes persisted?

- [ ] if qset for an incoming validation request is missing, should we wait until it arrives from discovery (instead of saying invalid)? Dos?

- restarting scp
    - [ ] Feed all persisted envelopes above last committed
    - [ ] A validate request for an already nominated block may result in missing txs, so don’t nominated before persisting txs (and serve them on demand) A node may have to request missing txs from self (if it nominated before restart)
    - [ ] Coordinator also needs to remember it already nominated?
    - [ ] unit tests?

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
