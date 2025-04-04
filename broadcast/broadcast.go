package broadcast

type broadcastAPI struct {
	svc *BroadcastService
}

func (s *broadcastAPI) AdvanceEpoch(delta int) Epoch {
	panic("not implemented")
}

func (s *broadcastAPI) MakeChannel(epoch Epoch) SendChannel {

	s.svc.lk.Lock()
	defer s.svc.lk.Unlock()

	if epoch < s.svc.epoch {
		panic("new channel epoch must be no smaller than system epoch")
	}

	channelID := s.svc.generateChannelID()
	tree, children, treeHash := s.svc.buildMulticastTree(channelID)
	ch := Channel{
		Source_id:  s.svc.ValidatorPublicInfo().Public_key,
		Channel_id: channelID,
		Epoch:      epoch,
		Tree_hash:  treeHash,
	}
	sch := newSendChannel(s.svc, ch, tree, children)

	s.svc.sendCh[ch] = sch
	s.svc.trees[treeHash] = tree

	return sch
}

func (svc *BroadcastService) generateChannelID() ChannelID {
	svc.randLk.Lock()
	defer svc.randLk.Lock()
	var chID ChannelID
	svc.rand.Read(chID[:])
	return chID
}

func (svc *BroadcastService) buildMulticastTree(chID ChannelID) (MulticastTreeNodes, []ValidatorPublicInfo, MulticastTreeHash) {
	panic("not implemented")
}

func (s *broadcastAPI) AcceptChannel() ReceiveChannel {
	panic("not implemented")
}
