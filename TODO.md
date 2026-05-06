- [ ] churner with chosen number of restarting nodes
  - [ ] long-running ci with 2 non-restarting nodes

- [ ] bug: in churner experiment one node gets stuck at block 1150 (needs reproduction)

CORRECTNESS

- [ ] baton should not send nominations for slots it has nominated for already via set_state_from_envelope

- [ ] scp: do externalize messages even need validation?
- [ ] allow only messages from the transitive closure of our fba

- [ ] stellar: are future nomination envelopes not given to scp?
- [ ] stellar: is maybe valid response only given to ballot validations?

SECURITY

- [ ] only process scp envelopes coming from the transitive closure of our quorum set

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
