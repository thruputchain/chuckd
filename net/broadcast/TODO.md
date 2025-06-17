EFFICIENCY
- [ ] process deliveries using a pool of workers
  - [ ] add flag to customize
- [?] sign/don't sign over payload in bulk
- [ ] marshal/sign procedures should use bufferpools dependent on the context

- [ ] dont include previous message hash

- [ ] consider using unreliable broadcast (quic datagram) and still error correct across trees/shreds
- [ ] send multicast tree in first relayed message (if intermediate node)
- [ ] recycle multicast trees
  - [ ] enable out of order delivery (after processing for validity)
- [ ] expire/cancel backed up deliveries in queue

TEST AND BENCHMARK
- [ ] test flood mode
- [ ] aws benchmark
