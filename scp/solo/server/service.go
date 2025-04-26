package server

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/scp/solo/dial"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type SoloServerService struct {
	net         libnet.Network
	handler     api.SCPHandler
	slotTimeout time.Duration
	self        proto.ValidatorPrivateInfo
	quorum      []proto.ValidatorPublicInfo
	lk          sync.Mutex
	slot        map[proto.SoloSlotIndex]*slot
}

func NewSoloServerService(
	ctx context.Context,
	n libnet.Network,
	h api.SCPHandler,
	slotTimeout time.Duration,
	self proto.ValidatorPrivateInfo,
	quorum []proto.ValidatorPublicInfo,
) *SoloServerService {
	return &SoloServerService{
		net:         n,
		handler:     h,
		slotTimeout: slotTimeout,
		self:        self,
		quorum:      quorum,
		slot:        map[proto.SoloSlotIndex]*slot{},
	}
}

func (svc *SoloServerService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.SoloServer1_Server{Srv: svc})
}

func (svc *SoloServerService) Soloserver_null(context.Context) error {
	return nil
}

func (svc *SoloServerService) Soloserver_nominate(ctx context.Context, nomination proto.SoloSignedNomination) error {

	// check signatures
	if !protoutil.VerifyXdr(nomination.Signer.Public_key, &nomination.Nomination, nomination.Signature) {
		return nil
	}

	svc.lk.Lock()
	defer svc.lk.Unlock()
	slotIndex := nomination.Nomination.Slot_index
	s, ok := svc.slot[slotIndex]
	if !ok {
		s = newSlot(svc, slotIndex, svc.slotTimeout)
		svc.slot[slotIndex] = s
	}
	s.nominate(ctx, nomination.Nomination)
	return nil
}

func (svc *SoloServerService) externalize(ctx context.Context, slotIndex proto.SoloSlotIndex, value proto.SoloValue) {

	var wg sync.WaitGroup
	for _, v := range svc.quorum {
		wg.Add(1)
		go func(v proto.ValidatorPublicInfo) {
			defer wg.Done()

			// XXX: use cached/persistent connections
			c, err := dial.DialSoloClientClient(ctx, svc.net, v.Address)
			if err != nil {
				log.Errorf("solo node %s not reachable for externalization: %s", protoutil.ValidatorPublicShortString(v), err)
				return
			}

			externalized := proto.SoloExternalized{
				Slot_index: slotIndex,
				Value:      value,
			}
			signed := proto.SoloSignedExternalized{
				Signer:       svc.self.Public_info,
				Signature:    protoutil.SignXdr(svc.self.Secret_key, &externalized),
				Externalized: externalized,
			}

			err = c.Soloclient_externalized(ctx, signed)
			if err != nil {
				log.Errorf("solo externalization error from %s: %s", protoutil.ValidatorPublicShortString(v), err)
				return
			}
		}(v)
	}
	wg.Wait()
}
