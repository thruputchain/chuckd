package discovery

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// TODO: share our quorums, when?

type DiscoveryService struct {
	lk      sync.Mutex
	vslices map[proto.ValidatorID]proto.ValidatorQuorumSlicesSnapshot
}

func NewDiscoveryService(ctx context.Context) *DiscoveryService {
	return &DiscoveryService{
		vslices: map[proto.ValidatorID]proto.ValidatorQuorumSlicesSnapshot{},
	}
}

func (s *DiscoveryService) Discovery_null() {}

func (s *DiscoveryService) Discovery_update(vs proto.ValidatorQuorumSlicesSnapshots) {
	for _, v := range vs {
		if !protoutil.Verify_ValidatorQuorumSlicesSnapshot(v) {
			continue
		}
		s.lk.Lock()
		s.vslices[protoutil.ValidatorPublicKeyToID(v.Validator_public_info.Public_key)] = v
		s.lk.Unlock()
	}
}
