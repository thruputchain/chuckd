package broadcast

func (s *broadcastAPI) AcceptChannel() ReceiveChannel {
	return <-s.svc.accept
}
