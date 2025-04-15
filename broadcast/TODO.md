- malicious node can replay a delivery to dos receiver or source
  - [ ] remedy: request tree from source
  - [ ] include source validator public info in the message
  - source signs the channel to bind tree hash with other channel info
  - verify hash integrity of trees received over the wire

[ ] XXX: what happens if an rpc call fails or timeouts?

[ ] reconciliation
- missing msg structure use msg hashes (not seqno)
- expose msg hashes to app

[ ] striped broadcasts
