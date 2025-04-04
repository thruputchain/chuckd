package broadcast

type recvChannel struct {
	svc       *BroadcastService
	channel   Channel
	treeHash  MulticastTreeHash
	treeNodes *MulticastTreeNodes
	children  *[]ValidatorPublicInfo
	received  []Message
}

func newRecvChannel(
	svc *BroadcastService,
	channel Channel,
	treeHash MulticastTreeHash,
	treeNodes *MulticastTreeNodes,
) *recvChannel {
	return &recvChannel{
		svc:       svc,
		channel:   channel,
		treeHash:  treeHash,
		treeNodes: treeNodes,
	}
}

func (x *recvChannel) Channel() Channel {
	return x.channel
}

func (x *recvChannel) Epoch() Epoch {
	return x.channel.Epoch
}

func (x *recvChannel) Receive() (seqno SeqNo, payload []byte, err error) {
	panic("not implemented")
}
