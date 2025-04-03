package broadcast

import proto "github.com/groundhogchain/protocol-go"

func (s *BroadcastService) findTree(h MulticastTreeHash) MulticastTreeNodes {
	s.lk.Lock()
	defer s.lk.Unlock()
	return s.trees[h]
}

func (s *BroadcastService) enqueueDelivery(v proto.ValidatorPublicInfo, delivery proto.Delivery) {
	s.deliveryQueue.Enqueue(v, delivery)
}
