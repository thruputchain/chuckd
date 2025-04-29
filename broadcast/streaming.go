package broadcast

import "context"

func (s *broadcastAPI) MakeChannel(ctx context.Context) SendChannel {

	s.svc.lk.Lock()
	defer s.svc.lk.Unlock()

	channelID := s.svc.generateChannelID()
	treeNodes, children, treeHash := s.svc.buildMulticastTree(channelID)
	ch := Channel{
		Source_id:  s.svc.ValidatorID(),
		Channel_id: channelID,
		Tree_hash:  treeHash,
		Batch_size: -1, // this is a streaming channel
	}
	sch := newSending(ctx, s.svc, ch, treeNodes, children)

	s.svc.sending[ch] = sch
	s.svc.trees.Ref(treeHash, treeNodes)

	return sch
}

func (s *broadcastAPI) AcceptChannel() ReceiveChannel {
	return <-s.svc.acceptCh
}
