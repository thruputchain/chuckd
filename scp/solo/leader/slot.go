package leader

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/scp/solo/dial"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type vhash = [32]byte // hash of nomoinated value

type slot struct {
	started    time.Time
	svc        *SoloLeaderService
	index      proto.SoloSlotIndex
	delay      time.Duration
	lk         sync.Mutex
	validated  map[vhash]map[proto.ValidatorID]bool
	candidates []proto.SoloValue
}

func newSlot(svc *SoloLeaderService, index proto.SoloSlotIndex, slotDelay time.Duration) *slot {
	return &slot{
		started:   time.Now(),
		svc:       svc,
		index:     index,
		delay:     slotDelay,
		validated: map[vhash]map[proto.ValidatorID]bool{},
	}
}

func (s *slot) nominate(ctx context.Context, nomination proto.SoloNomination) {
	// ask every node to validate the block
	h := sha256.Sum256(nomination.Value)
	go func() {
		var wg sync.WaitGroup
		for _, v := range s.svc.quorum {
			wg.Add(1)
			go func(v proto.ValidatorPublicInfo) {
				defer wg.Done()

				c, err := dial.DialSoloFollowerClient(ctx, s.svc.net, v.Address)
				if err != nil {
					log.Errorf("solo node %s not reachable for validation at %s: %s", protoutil.ValidatorPublicShortString(v), v.Address, err)
					return
				}

				validate := proto.SoloValidate{
					Slot_index: nomination.Slot_index,
					Value:      nomination.Value,
				}
				signed := proto.SoloSignedValidate{
					Signer:    s.svc.self.Public_info,
					Signature: protoutil.SignXdr(s.svc.self.Secret_key, &validate),
					Validate:  validate,
				}

				valid, err := c.Solofollower_validate(ctx, signed)
				if err != nil {
					log.Errorf("solo validation error from %s: %s", protoutil.ValidatorPublicShortString(v), err)
					return
				}
				if valid != proto.SoloFullyValidValue {
					return
				}

				// update validated record
				s.lk.Lock()
				defer s.lk.Unlock()

				vid := protoutil.VIDOfPublic(v)
				hv, ok := s.validated[h]
				if !ok {
					hv = map[proto.ValidatorID]bool{}
					s.validated[h] = hv
				}
				hv[vid] = true

			}(v)
		}
		wg.Wait()

		// update candidates for slot
		s.lk.Lock()
		defer s.lk.Unlock()

		if len(s.validated[h]) >= len(s.svc.quorum)/2+1 { // confirm on majority
			s.candidates = append(s.candidates, nomination.Value)
		}
	}()
}

func (svc *SoloLeaderService) flush(ctx context.Context) {
	for {
		time.Sleep(svc.slotDelay)
		now := time.Now()

		// externalize and garbage collect slots
		svc.lk.Lock()
		var readySlots []*slot
		for _, slot := range svc.slot {
			if len(slot.candidates) > 0 && now.Sub(slot.started) > slot.delay {
				readySlots = append(readySlots, slot)
				delete(svc.slot, slot.index)
			}
		}
		svc.lk.Unlock()
		for _, slot := range readySlots {
			combined := svc.handler.CombineCandidates(ctx, slot.index, slot.candidates)
			svc.externalize(ctx, slot.index, combined)
		}
	}
}
