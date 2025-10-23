- [ ] purpose of extract valid value?
- [ ] purpose of MaybeValid?

CONSENSUS SCP VALUE SEPARATION

- [ ] baton
  - [ ] notifies cvpool of new externalizations
  - [ ] send nomination to cvpool and disseminates nominations _after_ scp nominate approves them

- [ ] system
  - [ ] receive cvs into cvpool

- [ ] repair
  - [x] rpc for cv
  - [x] agent for cv

- [ ] use ConsensusValueHash as SCPValue

- [ ] scp validate exposes SCPNodeID (so repair knows where to look)

UPGRADES

- [ ] verify that catchup blocks (proposal+upgrades) include valid upgrades
  - [ ] current block hash linking algorithm does not consider upgrades

- [ ] baton pulls pending upgrades from persistence and places them in consensus values
- [ ] extract valid value strips unrecognizable upgrades?
- [ ] NOTE that baton.validated caches block validations, but on the upgrade validations
- [ ] interaction between storage and protocol upgrades???

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
