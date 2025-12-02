SCHEDULER INTEGRATION

HOGSCHED

- [ ] support available host allocation
  - [ ] unit test lock service
  - [x] set alias map HOG_HOSTVAR_XXX="hog7", etc. for all jobs
  - [x] rewrite spec with assigned hosts

- [x] interrupt hog_sched kills subprocesses?

- [ ] when running hogd as user, cargo builds are conflicting with each other:
```
[hog6:out] error: failed to download `tracing v0.1.43`
[hog6:out]
[hog6:out] Caused by:
[hog6:out]   unable to get packages from source
[hog6:out]
[hog6:out] Caused by:
[hog6:out]   failed to unpack package `tracing v0.1.43`
[hog6:out]
[hog6:out] Caused by:
[hog6:out]   failed to open `/tmp/hogjob-petar/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/tracing-0.1.43/.cargo-ok`
[hog6:out]
[hog6:out] Caused by:
[hog6:out]   File exists (os error 17)
```

- nspawn
  - networking with fixed ip
  - forwarding ssh agent

    sudo systemd-nspawn -D /var/lib/machines/petar-hogbase \
    --bind="$SSH_AUTH_SOCK:/tmp/ssh-agent.sock" \
    --setenv=SSH_AUTH_SOCK=/tmp/ssh-agent.sock \
    --network-zone=lan \
    --network-interface=enp129s0f0np0:eth0


BUGS

- [ ] coordinator_bench fails when threshold=2
  - [ ] one v gets stuck on `missingcv.go:63 no repaired consensus value returned`
  - [ ] can repair missing txs be stuck in a similar way?

- [ ] put height in cvc


  - CAUSE: cv is dropped when peers commit it, and thus current peer fails to repair cv
  - Q: why is the node stuck and not seeing higher externalizations?
    - A: because externalizations come from SCP, but SCP cannot process an externalization before fetching the CV from the CVC

  - GOAL: XXX

  -> APPROACH 1:
    - put height in cvc, and repair extracts cv from storage using height 
    - unblocks externalizations to push through scp and reach baton
  - APPROACH 2:
    - index cvs by hash in storage layer


- [ ] why `dogwood dropping partial incomplete transmission` appears before `counterparties online`









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

UPDATE QUORUM SET

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
