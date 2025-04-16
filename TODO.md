[ ] handle RPC call errors everywhere

GITHUB
- [ ] code coverage
- [ ] code builds and tests

CODE QUALITY
- [ ] renames
    - Multicast->M
    - Validator->V
    - Quorum->Q

GOXDR
- [ ] ctx on xdr methods?
- [ ] plumb ctx in rpc driver.Go, which supports context abort

DUE DILLIGENCE
- [ ] place all constants into configuration structures
- [ ] timeouts on all rpc calls
- [ ] generate RPC clients that return errors (instead of panicking)
