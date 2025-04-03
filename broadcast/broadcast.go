package broadcast

import proto "github.com/groundhogchain/protocol-go"

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
	ch := proto.BroadcastChannel{
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
	panic("not implemented")
}

func (svc *BroadcastService) buildMulticastTree(chID ChannelID) (MulticastTreeNodes, []proto.ValidatorPublicInfo, MulticastTreeHash) {
	panic("not implemented")
}

func (s *broadcastAPI) AcceptChannel() ReceiveChannel {
	panic("not implemented")
}
