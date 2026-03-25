CATCHUP BLOCK PERSISTENCE

- [ ] loadgen test catchup (3-node, 1 node restarts)

SCP

- [x] are our scp envelopes persisted?

- [x] scp envelope persistence should NOT add timestamp (multiplies refeeded messages)
- [ ] compress scp envelopes using their monotone order

- [ ] must we serve missing txs for our nominations after they are committed?
  - dm: You need to keep transactions around for a little while, yes. Imagine a symmetric configuration with a quorum size of 2f+1 out of 3f+1 nodes. Now say f nodes are slow and missing the previous block, but then a different f nodes crash permanently. You don't want to have to do f complete blockchain state transfers when it would suffice just to send the f slow nodes a block's worth of transactions.

- [ ] persist open nominations
  - [ ] on restart submit_vetted their txs to scheduler

===

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
