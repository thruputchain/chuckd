- [ ] purpose of extract valid value?
- [ ] purpose of MaybeValid?

CONSENSUS SCP VALUE SEPARATION

- [x] baton
  - [x] notifies cvpool of new externalizations
  - [x] use ConsensusValueCue as SCPValue
    - [x] nomination
      - [x] send nomination to cvpool and disseminates nominations _after_ scp nominate approves them
      - [x] nominate returns error on rejection and this is handled in driver
      - [x] impl DisseminateCV in live
      - [x] impl DisseminateCV in test
    - [x] validation
      - [x] add block height to cue? (no, use slot instead)
      - [x] add block hash to cue? (no, cache validated value hashes)
      - [x] cache validated cvs by hash
      - [x] add proposer to cue, and signature to enable slashing (no, signature can be made available from scp)
      - [x] plumb handler methods
      - [x] add invalidated cache, purge appropriately
    - [x] externalization

- [x] system
  - [x] drive received cvs into cvpool

- [x] repair
  - [x] rpc for cv
  - [x] agent for cv

- [ ] test
  - [x] unit test
    - [x] add unit test for validations below externalized
  - [ ] run eval
    - [ ] 3 nodes
    - [ ] 1 node

- [ ] SUPPORT VALIDATING PAST EXTERNALIZATIONS
  - [x] if validation request is for an already externalized slot, validate based on saved externalizations
  - [x] when context is cancelled due to newer externalization, validation has to look at the externalized value and make a judgement based on that
  - [x] validated and invalidated structures should save the cue to enable checking the whole cue value for correctness
  - [ ] check if we can fix purge bottom in scp server to 1 below externalized; handler.rs:541


- [ ] scp validate exposes SCPNodeID (so repair knows where to look)

SCP ENVELOPE ACTIVE REPAIR 

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
