package broadcast

type receiveChannel struct {
	channelID ChannelID
	epoch     Epoch
}

func (x *receiveChannel) ChannelID() ChannelID {
	return x.channelID
}

func (x *receiveChannel) Epoch() Epoch {
	return x.epoch
}

func (x *receiveChannel) Receive() (seqno SeqNo, payload []byte, err error) {
	panic("not implemented")
}
