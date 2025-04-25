package server

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libnet"
	proto "github.com/groundhogchain/protocol-go"
)

type SoloServerService struct {
	net    libnet.Network
	rand   *rand.Rand
	self   proto.ValidatorPrivateInfo
	quorum []proto.ValidatorPublicInfo
	lk     sync.Mutex
	slot   map[proto.SoloSlotIndex]*slot
}

func NewSoloServerService(
	ctx context.Context,
	n libnet.Network,
	r *rand.Rand,
	self proto.ValidatorPrivateInfo,
	quorum []proto.ValidatorPublicInfo,
) *SoloServerService {
	return &SoloServerService{
		net:    n,
		rand:   r,
		self:   self,
		quorum: quorum,
		slot:   map[proto.SoloSlotIndex]*slot{},
	}
}

func (svc *SoloServerService) Soloserver_null(context.Context) error {
	return nil
}

func (svc *SoloServerService) Soloserver_nominate(ctx context.Context, nomination proto.SoloSignedNomination) error {

	// XXX // check signatures

	svc.lk.Lock()
	defer svc.lk.Unlock()
	slotIndex := nomination.Nomination.Slot_index
	s, ok := svc.slot[slotIndex]
	if !ok {
		s = newSlot(svc, slotIndex)
		svc.slot[slotIndex] = s
	}
	s.nominate(ctx, nomination.Nomination)
	return nil
}
