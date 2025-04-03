package broadcast

import (
	"context"

	"github.com/groundhogchain/chuckd/discovery"
	proto "github.com/groundhogchain/protocol-go"
)

type BroadcastService struct {
	privInfo  proto.ValidatorPrivateInfo
	ourSlices []proto.QuorumSlice
	discovery *discovery.DiscoveryService
}

func NewBroadcastService(
	ctx context.Context,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	discovery *discovery.DiscoveryService,
) *BroadcastService {

	s := &BroadcastService{
		privInfo:  privInfo,
		ourSlices: ourSlices,
		discovery: discovery,
	}

	return s
}

func (s *BroadcastService) Broadcast_null() {}

func (s *BroadcastService) Broadcast_deliver(x proto.Delivery) {
	panic("not implemented")
}

func (s *BroadcastService) Broadcast_reconcile(x proto.ReconcileMessages) proto.ReconciledMessages {
	panic("not implemented")
}

func (s *BroadcastService) Broadcast_expand(x proto.MulticastTreeHash) proto.ExpandedMulticastTree {
	panic("not implemented")
}
