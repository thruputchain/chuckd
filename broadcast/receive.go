package broadcast

type recvChannel struct {
	svc     *BroadcastService
	channel Channel
}

func newRecvChannel(svc *BroadcastService, channel Channel) *recvChannel {
	return &recvChannel{svc: svc, channel: channel}
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
