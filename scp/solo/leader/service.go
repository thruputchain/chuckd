package leader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/scp/solo/dial"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type SoloLeaderService struct {
	ctx       context.Context
	net       libnet.Network
	handler   scpapi.SCPHandler
	slotDelay time.Duration
	self      proto.ValidatorPrivateInfo
	quorum    []proto.ValidatorPublicInfo
	m         xdrutil.XdrMarshaler
	lk        sync.Mutex
	slot      map[proto.SoloSlotIndex]*slot
}

func NewSoloLeaderService(
	ctx context.Context,
	n libnet.Network,
	h scpapi.SCPHandler,
	// slots don't externalize before the timeout to allow room for competing nominations
	// slotDelay should be much smaller, say 1/12, than overall externalization time
	slotDelay time.Duration,
	self proto.ValidatorPrivateInfo,
	quorum []proto.ValidatorPublicInfo,
) *SoloLeaderService {
	svc := &SoloLeaderService{
		ctx:       ctx,
		net:       n,
		handler:   h,
		slotDelay: slotDelay,
		self:      self,
		quorum:    quorum,
		m:         xdrutil.NewXdrMarshaler(),
		slot:      map[proto.SoloSlotIndex]*slot{},
	}
	go svc.flush(ctx)
	return svc
}

func (svc *SoloLeaderService) Start(ctx context.Context, address string) {
	rpcutil.StartSerial(ctx, svc.net, address, proto.SoloLeader1_Server{Srv: svc})
}

func (svc *SoloLeaderService) Sololeader_null(context.Context) error {
	return nil
}

func (svc *SoloLeaderService) Sololeader_nominate(ctx context.Context, nomination proto.SoloSignedNomination) error {

	liblog.Infof("solo leader received nomination for slot %d from %v",
		nomination.Nomination.Slot_index, protoutil.VIDString(protoutil.VIDOfPublic(nomination.Signer)))

	// check signatures
	if !protoutil.VerifyXdr(svc.m, nomination.Signer.Public_key, &nomination.Nomination, nomination.Signature) {
		return fmt.Errorf("bad signature")
	}

	svc.lk.Lock()
	defer svc.lk.Unlock()
	slotIndex := nomination.Nomination.Slot_index
	s, ok := svc.slot[slotIndex]
	if !ok {
		s = newSlot(svc, slotIndex, svc.slotDelay)
		svc.slot[slotIndex] = s
	}
	s.nominate(ctx, nomination.Nomination)
	return nil
}

func (svc *SoloLeaderService) externalize(ctx context.Context, slotIndex proto.SoloSlotIndex, value proto.SoloValue) {
	liblog.Infof("solo leader externalizing slot %d", slotIndex)

	var wg sync.WaitGroup
	for _, v := range svc.quorum {
		wg.Add(1)
		go func(v proto.ValidatorPublicInfo) {
			defer wg.Done()

			c, err := dial.DialSoloFollowerClient(ctx, svc.net, v.Address)
			if err != nil {
				liblog.Errorf("solo follower %s at %s not reachable for externalization: %s", protoutil.VPublicShortString(v), v.Address, err)
				return
			}
			defer c.Close()

			externalized := proto.SoloExternalized{
				Slot_index: slotIndex,
				Value:      value,
			}
			signed := proto.SoloSignedExternalized{
				Signer:       svc.self.Public_info,
				Signature:    protoutil.SignXdr(svc.m, svc.self.Secret_key, &externalized),
				Externalized: externalized,
			}

			err = c.Solofollower_externalized(ctx, signed)
			if err != nil {
				liblog.Errorf("solo externalization error from %s: %s", protoutil.VPublicShortString(v), err)
				return
			}
		}(v)
	}
	wg.Wait()
}
