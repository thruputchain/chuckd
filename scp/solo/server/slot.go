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
					log.Errorf("solo node %s not reachable: %s", protoutil.ValidatorPublicShortString(v), err)
					return
				}

				valid, err := c.Soloclient_validate(ctx, signed)
				if err != nil {
					log.Errorf("solo validation error from %s: %s", protoutil.ValidatorPublicShortString(v), err)
					return
				}
				if !valid {
					return
				}

				s.lk.Lock()
				defer s.lk.Unlock()
				s.validated[protoutil.ValidatorIDFromPublic(v)] = true
			}(v)
		}
		wg.Wait()

		XXX // externalize and garbage collect
	}()
}
