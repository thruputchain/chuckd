[x] payload and route transmission on different paths
[x] process receptions even if route not known

INVESTIGATE
[ ] check for memory leaks in dogwood
[ ] why is quic slower than tcp

[ ] understand how many cores reedsolomon uses
[ ] shuffle code chunks

[x] protoutil sign/verify over hash of message (1.7GB/sec)
[ ] unit test does not work with QUIC???

[ ] rpc improvements (caller won't need a worker pool to fire calls on one goroutine)
    [ ] client: don't wait for reply
    [ ] server: don't reply

[?] expire deliveries waiting in send queue

[ ] vary code parameters based on message size
[ ] transmission affinity/priority
