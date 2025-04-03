package broadcast

type broadcastAPI struct {
	service *BroadcastService
}

func (s *broadcastAPI) AdvanceEpoch(delta int) Epoch {
	panic("not implemented")
}

func (s *broadcastAPI) MakeChannel(Epoch) SendChannel {
	panic("not implemented")
}

func (s *broadcastAPI) AcceptChannel() ReceiveChannel {
	panic("not implemented")
}
