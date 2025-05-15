package leader

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

type SoloLeaderService struct {
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
	// slots don't externalize before the timeout to allow room for competing nominations
	// slotTimeout should be much smaller, say 1/12, than overall externalization time
	slotTimeout time.Duration,
	self proto.ValidatorPrivateInfo,
	quorum []proto.ValidatorPublicInfo,
) *SoloLeaderService {
	svc := &SoloLeaderService{
		net:         n,
		handler:     h,
		slotTimeout: slotTimeout,
		self:        self,
		quorum:      quorum,
		slot:        map[proto.SoloSlotIndex]*slot{},
	}
	go svc.flush(ctx)
	return svc
}

func (svc *SoloLeaderService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.SoloLeader1_Server{Srv: svc})
}

func (svc *SoloLeaderService) Sololeader_null(context.Context) error {
	return nil
}

func (svc *SoloLeaderService) Sololeader_nominate(ctx context.Context, nomination proto.SoloSignedNomination) error {

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

func (svc *SoloLeaderService) externalize(ctx context.Context, slotIndex proto.SoloSlotIndex, value proto.SoloValue) {
	log.Infof("solo externalizing slot %d", slotIndex)

	var wg sync.WaitGroup
	for _, v := range svc.quorum {
		wg.Add(1)
		go func(v proto.ValidatorPublicInfo) {
			defer wg.Done()

			// XXX: use cached/persistent connections
			c, err := dial.DialSoloFollowerClient(ctx, svc.net, v.Address)
			if err != nil {
				log.Errorf("solo follower %s not reachable for externalization: %s", protoutil.ValidatorPublicShortString(v), err)
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

			err = c.Solofollower_externalized(ctx, signed)
			if err != nil {
				log.Errorf("solo externalization error from %s: %s", protoutil.ValidatorPublicShortString(v), err)
				return
			}
		}(v)
	}
	wg.Wait()
}
