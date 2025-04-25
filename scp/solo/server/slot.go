package server

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/scp/solo/dial"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type slot struct {
	svc       *SoloServerService
	index     proto.SoloSlotIndex
	lk        sync.Mutex
	validated map[proto.ValidatorID]bool
}

func newSlot(svc *SoloServerService, index proto.SoloSlotIndex) *slot {
	return &slot{
		svc:       svc,
		index:     index,
		validated: map[proto.ValidatorID]bool{},
	}
}

func (s *slot) nominate(ctx context.Context, nomination proto.SoloNomination) {
	// ask every node to validate the block
	go func() {
		var wg sync.WaitGroup
		for _, v := range s.svc.quorum {
			wg.Add(1)
			go func(v proto.ValidatorPublicInfo) {
				defer wg.Done()

				// XXX: use cached/persistent connections
				c, err := dial.DialSoloClientClient(ctx, s.svc.net, v.Address)
				if err != nil {
					log.Errorf("solo node %s not reachable for validation: %s", protoutil.ValidatorPublicShortString(v), err)
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

				valid, err := c.Soloclient_validate(ctx, signed)
				if err != nil {
					log.Errorf("solo validation error from %s: %s", protoutil.ValidatorPublicShortString(v), err)
					return
				}
				if valid != proto.SoloFullyValidValue {
					return
				}

				s.lk.Lock()
				defer s.lk.Unlock()
				s.validated[protoutil.ValidatorIDFromPublic(v)] = true
			}(v)
		}
		wg.Wait()

		// externalize and garbage collect
		s.lk.Lock()
		confirmed := len(s.validated) >= len(s.validated)/2+1
		s.lk.Unlock()

		s.svc.lk.Lock()
		// if a minority of malicious nodes try to revive the slot, they will fail for lack of majority validation.
		delete(s.svc.slot, s.index)
		s.svc.lk.Unlock()

		if confirmed {
			s.svc.externalize(ctx, nomination.Slot_index, nomination.Value)
		}
	}()
}
