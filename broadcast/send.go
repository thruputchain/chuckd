package broadcast

type sendChannel struct {
	channelID ChannelID
	epoch     Epoch
}

func (x *sendChannel) ChannelID() ChannelID {
	return x.channelID
}

func (x *sendChannel) Epoch() Epoch {
	return x.epoch
}

func (x *sendChannel) Send(payload []byte) {
	panic("not implemented")
}
