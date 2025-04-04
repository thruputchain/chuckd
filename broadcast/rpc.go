package broadcast

func (s *BroadcastService) Broadcast_null() {}

func (s *BroadcastService) Broadcast_deliver(x Delivery) {

	// channel := x.Msg.Header.Channel
	// check channel epoch
	// check if channel known

	panic("not implemented") // XXX: next up
}

func (s *BroadcastService) Broadcast_reconcile(x ReconcileMessages) ReconciledMessages {
	panic("not implemented")
}

func (s *BroadcastService) Broadcast_expand(x MulticastTreeHash) MulticastTreeNodes {
	return s.findTree(x)
}
