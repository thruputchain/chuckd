package broadcast

type receiveChannel struct{}

func (x *receiveChannel) ChannelID() ChannelID {
	panic("not implemented")
}

func (x *receiveChannel) Epoch() Epoch {
	panic("not implemented")
}

func (x *receiveChannel) Receive() (seqno SeqNo, payload []byte, err error) {
	panic("not implemented")
}
