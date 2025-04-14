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
	disc      AcceptDiscriminator
	//
	randLk sync.Mutex
	rand   *rand.Rand
	//
	lk        sync.Mutex
	sending   map[Channel]*sending
	receiving map[Channel]*receiving
	accepting map[Channel]*accepting
	trees     map[MulticastTreeHash]MulticastTreeNodes // cached multicast trees
	//
	deliveryQueue *DeliveryQueue
	//
	acceptCh chan *receiving
}

func NewBroadcastService(
	ctx context.Context,
	rr *rand.Rand,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	router router.Router,
	disc AcceptDiscriminator,
) *BroadcastService {
	log.Infof("starting broadcast service")
	s := &BroadcastService{
		privInfo:  privInfo,
		ourSlices: ourSlices,
		router:    router,
		disc:      disc,
		//
		sending:   map[Channel]*sending{},
		receiving: map[Channel]*receiving{},
		accepting: map[Channel]*accepting{},
		trees:     map[MulticastTreeHash]MulticastTreeNodes{},
		//
		deliveryQueue: NewDeliveryQueue(ctx),
		//
		acceptCh: make(chan *receiving),
	}
	return s
}

type broadcastAPI struct {
	svc *BroadcastService
}

func (svc *BroadcastService) API() Broadcast {
	return &broadcastAPI{svc: svc}
}

func (svc *BroadcastService) ValidatorPublicInfo() ValidatorPublicInfo {
	return svc.privInfo.Public_info
}

func (svc *BroadcastService) ValidatorID() proto.ValidatorID {
	return protoutil.ValidatorIDFromPublic(svc.ValidatorPublicInfo())
}
