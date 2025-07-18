[x] payload and route transmission on different paths
[x] process receptions even if route not known

OPEN
[ ] measure wait for worker routines
[ ] measure client pool utilization

[ ] should rpc MsgPool be shared across drivers and transports
[ ] should endpoint.Start not used a background loop?
[ ] why is quic slower than tcp


[ ] understand how many cores reed solomon uses
[ ] shuffle code chunks


[x] protoutil sign/verify over hash of message (1.7GB/sec)
[ ] unit test does not work with QUIC???

[ ] rpc improvements (caller won't need a worker pool to fire calls on one goroutine)
    [ ] client: don't wait for reply
    [ ] server: don't reply

[?] expire deliveries waiting in send queue

[ ] vary code parameters based on message size
[ ] transmission affinity/priority
