package broadcast

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type BroadcastService struct {
	privInfo  proto.ValidatorPrivateInfo
	ourSlices []proto.QuorumSlice
	router    router.Router
	//
	randLk sync.Mutex
	rand   *rand.Rand
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
	rr *rand.Rand,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	router router.Router,
) *BroadcastService {

	log.Infof("starting broadcast service")
	s := &BroadcastService{
		privInfo:  privInfo,
		ourSlices: ourSlices,
		router:    router,
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

func (s *BroadcastService) ValidatorPublicInfo() ValidatorPublicInfo {
	return s.privInfo.Public_info
}

func (s *BroadcastService) ValidatorID() proto.ValidatorID {
	return protoutil.ValidatorIDFromPublic(s.ValidatorPublicInfo())
}

func (s *BroadcastService) Broadcast_null() {}

func (s *BroadcastService) Broadcast_deliver(x Delivery) {
	panic("not implemented") // XXX: next up
}

func (s *BroadcastService) Broadcast_reconcile(x ReconcileMessages) ReconciledMessages {
	panic("not implemented")
}

func (s *BroadcastService) Broadcast_expand(x MulticastTreeHash) MulticastTreeNodes {
	return s.findTree(x)
}
