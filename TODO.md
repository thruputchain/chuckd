SCHEDULER INTEGRATION & PERF

43.32066288 BEGIN
    [ ] 32ms (nominate.go:87) first computation of cv hash; can change way hash is computed
43.352560498 hashed cv
    [x] 15ms (SCP calls into coordinator)
43.368486816 received externalization
    [x] 36ms (scp.go:92) deparallelize nomination value on externalization; pre-compute ns value in background
        [ ] delay ns computation until/if actually needed
43.405349934 persisting cv begins
    [x] 35ms (scp.go:99) marshaling cv to file (cannot be improved, bottlnecked by fs write speed?)
43.44038479 persisted cv
    0ms
43.440907924 END

[ ] make all marshal-write-to-file directly marshal-to-file
[ ] make all read-from-file-unmarshal directly unmarshal-from-file

SCP

- [ ] use SCP proto definitions from the SCP module (chuckd -> scp-go -> scp)
- [ ] purpose of MaybeValid?
- [ ] ask stellar folks how they deal with messages coming after an externalization

BUGS

- [x] `repaired consensus value hash mismatch` https://github.com/groundhogchain/chuckd/actions/runs/20147245728/job/57830875188
  - [ ] what if consensus value for a non-externalized block is missing and externalizations have gone beyond?
- [ ] validators must be able to respond to missing txs request even for past blocks

HOGTOOLS

- [ ] serde support in x23c

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

OPTIMIZATIONS

- [ ] parallelize block proposal, optimistically pipeline by proposing on top of all currently being proposed blocks
- [ ] ask scp whether we should bother nominating this round

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
