
- [x] call executors in parallel; how to allocate them?

- [ ] cache result of validating proposals during nomination; skip re-validation during commit, after externalization


- [x] restore db
- [x] nominate (starts after db)
  - [ ] can nil be passed as previous value to Nominate?; or can i simply use the previous block hash?
- [x] catchup
- [ ] handle scp
- [ ] validate and commit
  - [ ] garbage collect
  - [ ] mark committed txs

- [ ] implement catchup service!
- [ ] garbage collection
