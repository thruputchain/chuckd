- [ ] update hogx for no wallet

DOGWOOD SECURITY

- [ ] add payload type to every send
- [ ] init dogwood with size limit function based on payload type
- [ ] multiplexer (user of dogwood) must check that payload type matches what's inside
- [ ] update dogwood rpc
- [ ] rename DogwoodInt64 to DogwoodU64, etc

addresses:

- https://github.com/groundhogchain/chuckd/issues/82

TESTS

- [ ] use scp timeout to reduce number of nominations in the network
  - stats on how often nominations timeout
  - stats on number of nominations per round

- [ ] test restart right after nomination
- [ ] test memory and goroutine leaks

SCP

- [ ] persist open nominations
  - [ ] on restart submit_vetted their txs to scheduler

- [ ] check unions in scp envelopes are correct before passing to scpd (at parsing level?)

- [ ] ask scp whether we should bother nominating this round (optimization)

- [ ] if qset for an incoming validation request is missing, should we wait until it arrives from discovery (instead of saying invalid)? Dos?

- withhold scp envelopes with missing qsets until qsets learned or slot subsumed by externalizations?

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
