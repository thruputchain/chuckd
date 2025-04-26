package server

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

type vhash = [32]byte

type slot struct {
	started    time.Time
	svc        *SoloServerService
	index      proto.SoloSlotIndex
	timeout    time.Duration
	lk         sync.Mutex
	validated  map[vhash]map[proto.ValidatorID]bool
	candidates []proto.SoloValue
}

func newSlot(svc *SoloServerService, index proto.SoloSlotIndex, slotTimeout time.Duration) *slot {
	return &slot{
		started:   time.Now(),
		svc:       svc,
		index:     index,
		timeout:   slotTimeout,
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

				// update validated record
				s.lk.Lock()
				defer s.lk.Unlock()

				vid := protoutil.ValidatorIDFromPublic(v)
				hv, ok := s.validated[h]
				if !ok {
					hv = map[proto.ValidatorID]bool{}
					s.validated[h] = hv
				}
				hv[vid] = true
			}(v)
		}
		wg.Wait()

		// externalize and garbage collect
		s.lk.Lock()
		confirmed := len(s.validated[h]) >= len(s.validated[h])/2+1
		s.candidates = append(s.candidates, nomination.Value)
		s.lk.Unlock()

		// if confirmed and the round timer has expired, combine and externalize
		if confirmed {
			if time.Now().Sub(s.started) > s.timeout {

				s.svc.lk.Lock()
				// if a minority of malicious nodes tries to revive the slot,
				// they will fail for lack of majority validation.
				delete(s.svc.slot, s.index)
				s.svc.lk.Unlock()

				combined := s.svc.handler.CombineCandidates(s.index, s.candidates)
				s.svc.externalize(ctx, nomination.Slot_index, combined)
			}
		}
	}()
}
