package broadcast

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type BroadcastService struct {
	dialer   libnet.AddressDialer
	privInfo proto.ValidatorPrivateInfo
	router   router.Router
	disc     AcceptDiscriminator
	//
	randLk sync.Mutex
	rand   *rand.Rand
	//
	lk        sync.Mutex
	sending   map[Channel]*sending
	receiving map[Channel]*receiving
	accepting map[Channel]*accepting
	trees     *TreeCache
	//
	deliveryQueue *DeliveryQueue
	//
	acceptCh chan *receiving
}

func NewBroadcastService(
	ctx context.Context,
	dialer libnet.AddressDialer,
	rr *rand.Rand,
	privInfo proto.ValidatorPrivateInfo,
	router router.Router,
	disc AcceptDiscriminator,
) *BroadcastService {
	log.Infof("starting broadcast service for %s", protoutil.ValidatorPublicShortString(privInfo.Public_info))
	s := &BroadcastService{
		dialer:   dialer,
		privInfo: privInfo,
		router:   router,
		disc:     disc,
		//
		rand: rr,
		//
		sending:   map[Channel]*sending{},
		receiving: map[Channel]*receiving{},
		accepting: map[Channel]*accepting{},
		trees:     NewTreeCache(),
		//
		deliveryQueue: NewDeliveryQueue(ctx, dialer),
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

func (svc *BroadcastService) SecretKey() proto.ValidatorSecretKey {
	return svc.privInfo.Secret_key
}

func (svc *BroadcastService) ValidatorID() proto.ValidatorID {
	return protoutil.ValidatorIDFromPublic(svc.ValidatorPublicInfo())
}
