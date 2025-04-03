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
	sendCh map[Channel]bool                          // send channels that are still alive
	recvCh map[Channel]bool                          // known receive channels that are still alive
	trees  map[MulticastTreeHash][]MulticastTreeNode // cached multicast trees
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
		epoch:     0,
		sendCh:    map[Channel]bool{},
		recvCh:    map[Channel]bool{},
		trees:     map[MulticastTreeHash][]MulticastTreeNode{},
	}

	return s
}

func (s *BroadcastService) API() Broadcast {
	return &broadcastAPI{service: s}
}

func (s *BroadcastService) Broadcast_null() {}

func (s *BroadcastService) Broadcast_deliver(x proto.Delivery) {
	panic("not implemented")
}

func (s *BroadcastService) Broadcast_reconcile(x proto.ReconcileMessages) proto.ReconciledMessages {
	panic("not implemented")
}

func (s *BroadcastService) Broadcast_expand(x MulticastTreeHash) proto.ExpandedMulticastTree {
	return proto.ExpandedMulticastTree{Tree_nodes: s.findTree(x)}
}
