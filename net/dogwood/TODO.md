
[ ] understand how many cores reedsolomon uses
[ ] shuffle code chunks
[ ] vary code parameters based on message size

[ ] protoutil sign over hash of message (vs over message)?

[ ] worker pool for ingesting deliveries (signature verification and decoding)
    [ ] config parameter for pool size
    [ ] additional worker pool for decoding?
[ ] sign over hash of payload (not over payload)

[ ] rpc improvements (caller won't need a worker pool to fire calls on one goroutine)
    [ ] client: don't wait for reply
    [ ] server: don't reply

[?] expire deliveries waiting in send queue
