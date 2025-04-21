package broadcast

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (svc *BroadcastService) findTree(h MulticastTreeHash) MulticastTreeNodes {
	svc.lk.Lock()
	defer svc.lk.Unlock()
	nodes, _ := svc.trees.Find(h)
	return nodes
}

func (svc *BroadcastService) enqueueDelivery(ctx context.Context, v proto.ValidatorPublicInfo, delivery proto.Delivery) error {
	return svc.deliveryQueue.Enqueue(ctx, v, delivery)
}
