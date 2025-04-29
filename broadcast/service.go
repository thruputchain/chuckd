package broadcast

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type BroadcastService struct {
	cfg      BroadcastConfig
	net      libnet.Network
	privInfo proto.ValidatorPrivateInfo
	router   router.Router
	disc     AcceptDiscriminator
	//
	metrics *BroadcastMetrics
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
	streamingAcceptCh chan *receiving
	bulkAcceptCh      chan *receiving
}

func NewBroadcastService(
	ctx context.Context,
	cfg BroadcastConfig,
	n libnet.Network,
	rr *rand.Rand,
	privInfo proto.ValidatorPrivateInfo,
	router router.Router,
	disc AcceptDiscriminator,
) *BroadcastService {
	log.Infof("starting broadcast service for %s", protoutil.ValidatorPublicShortString(privInfo.Public_info))
	m := NewBroadcastMetrics(ctx)
	s := &BroadcastService{
		cfg:      cfg,
		net:      n,
		privInfo: privInfo,
		router:   router,
		disc:     disc,
		//
		metrics: m,
		//
		rand: rr,
		//
		sending:   map[Channel]*sending{},
		receiving: map[Channel]*receiving{},
		accepting: map[Channel]*accepting{},
		trees:     NewTreeCache(cfg.UnusedMulticastTreeTTL),
		//
		streamingAcceptCh: make(chan *receiving),
		bulkAcceptCh:      make(chan *receiving),
	}
	return s
}

func (svc *BroadcastService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.Broadcast1_Server{Srv: svc})
}

type broadcastAPI struct {
	svc *BroadcastService
}

func (svc *BroadcastService) StreamingAPI() StreamingBroadcast {
	return &broadcastAPI{svc: svc}
}

func (svc *BroadcastService) BulkAPI() BulkBroadcast {
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
