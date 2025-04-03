package broadcast

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/discovery"
	"github.com/groundhogchain/chuckd/lib/log"
	proto "github.com/groundhogchain/protocol-go"
)

type BroadcastService struct {
	privInfo  proto.ValidatorPrivateInfo
	ourSlices []proto.QuorumSlice
	discovery *discovery.DiscoveryService
	//
	lk     sync.Mutex
	epoch  Epoch
	sendCh map[Channel]*sendChannel                 // send channels that are still alive
	recvCh map[Channel]*recvChannel                 // known receive channels that are still alive
	trees  map[MulticastTreeHash]MulticastTreeNodes // cached multicast trees
	//
	deliveryQueue *DeliveryQueue
}

func NewBroadcastService(
	ctx context.Context,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	discovery *discovery.DiscoveryService,
) *BroadcastService {

	log.Infof("starting broadcast service")
	s := &BroadcastService{
		privInfo:  privInfo,
		ourSlices: ourSlices,
		discovery: discovery,
		//
		epoch:  0,
		sendCh: map[Channel]*sendChannel{},
		recvCh: map[Channel]*recvChannel{},
		trees:  map[MulticastTreeHash]MulticastTreeNodes{},
		//
		deliveryQueue: NewDeliveryQueue(ctx),
	}
	return s
}

func (s *BroadcastService) API() Broadcast {
	return &broadcastAPI{svc: s}
}

func (s *BroadcastService) ValidatorPublicInfo() proto.ValidatorPublicInfo {
	return s.privInfo.Public_info
}

func (s *BroadcastService) Broadcast_null() {}

func (s *BroadcastService) Broadcast_deliver(x proto.Delivery) {
	panic("not implemented") // XXX: next up
}

func (s *BroadcastService) Broadcast_reconcile(x proto.ReconcileMessages) proto.ReconciledMessages {
	panic("not implemented")
}

func (s *BroadcastService) Broadcast_expand(x MulticastTreeHash) MulticastTreeNodes {
	return s.findTree(x)
}
