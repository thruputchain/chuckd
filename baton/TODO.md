
- [x] call executors in parallel
- [x] cache result of validating proposals during nomination; skip re-validation during commit, after externalization
- [x] signal "nominate" when the db is up to speed, and we know the previous proposal
- [x] mark committed transactions when committing to db



- [x] restore db
- [x] nominate (starts after db)
  - [x] can nil be passed as previous value to Nominate?; or can i simply use the previous block hash?
- [x] catchup
- [ ] handle scp
- [ ] validate and commit
  - [ ] garbage collect
  - [ ] mark committed txs

- [ ] implement catchup service!
- [ ] garbage collection
