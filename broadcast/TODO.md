- [ ] test broadcast
  - [x] testing virtual network
  - [ ] test router

- [x] malicious node can replay a delivery to dos receiver or source
  - [x] include source validator public info in the message
  - [x] source signs the channel to bind tree hash with other channel info
    - [x] verify delivery signatures
  - [x] verify hash integrity of trees received over the wire

- [x] handle failed rpc calls
    - [x] broadcast
    - [x] discovery
    - [x] coord

- [x] reconciliation
  - [x] missing msg structure use msg hashes (not seqno)
  - [x] expose msg hashes to app

[ ] discovery service gossip sets of nodes (not slices)
    - https://github.com/stellar/stellar-xdr/blob/9ed933f5a7d4df9d3ef69a814a7c5756606ea7e7/Stellar-SCP.x#L80

[ ] striped broadcasts
