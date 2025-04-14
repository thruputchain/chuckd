package broadcast

import proto "github.com/groundhogchain/protocol-go"

func (s *BroadcastService) findTree(h MulticastTreeHash) MulticastTreeNodes {
	s.lk.Lock()
	defer s.lk.Unlock()
	nodes, _ := s.trees.Find(h)
	return nodes
}

func (s *BroadcastService) enqueueDelivery(v proto.ValidatorPublicInfo, delivery proto.Delivery) {
	s.deliveryQueue.Enqueue(v, delivery)
}
