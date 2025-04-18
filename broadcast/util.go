package broadcast

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (s *BroadcastService) findTree(h MulticastTreeHash) MulticastTreeNodes {
	s.lk.Lock()
	defer s.lk.Unlock()
	nodes, _ := s.trees.Find(h)
	return nodes
}

func (s *BroadcastService) enqueueDelivery(ctx context.Context, v proto.ValidatorPublicInfo, delivery proto.Delivery) error {
	return s.deliveryQueue.Enqueue(ctx, v, delivery)
}
