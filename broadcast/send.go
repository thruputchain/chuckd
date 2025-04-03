package broadcast

type sendChannel struct{}

func (x *sendChannel) ChannelID() ChannelID {
	panic("not implemented")
}

func (x *sendChannel) Epoch() Epoch {
	panic("not implemented")
}

func (x *sendChannel) Send(payload []byte) {
	panic("not implemented")
}
