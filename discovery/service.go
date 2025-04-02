package discovery

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type DiscoveryService struct {
	privInfo  proto.ValidatorPrivateInfo
	ourSlices []proto.QuorumSlice
	lk        sync.Mutex
	allSlices map[proto.ValidatorID]proto.ValidatorQuorumSlicesSnapshot
}

func NewDiscoveryService(
	ctx context.Context,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
) *DiscoveryService {

	s := &DiscoveryService{
		privInfo:  privInfo,
		ourSlices: ourSlices,
		allSlices: map[proto.ValidatorID]proto.ValidatorQuorumSlicesSnapshot{},
	}
	s.addQuorumSlices(protoutil.MakeValidatorQuorumSlicesSnapshot(privInfo, ourSlices))

	go s.advertise(ctx)
	return s
}

func (s *DiscoveryService) Discovery_null() {}

func (s *DiscoveryService) Discovery_update_quorums(vs proto.ValidatorQuorumSlicesSnapshots) proto.UpdateQuorumsResult {
	for _, v := range vs {
		s.addQuorumSlices(v)
	}
	panic("XXX")
}

func (s *DiscoveryService) addQuorumSlices(vslices proto.ValidatorQuorumSlicesSnapshot) {
	if !protoutil.VerifyValidatorQuorumSlicesSnapshot(vslices) {
		return
	}
	s.lk.Lock()
	s.allSlices[protoutil.ValidatorPublicKeyToID(vslices.Validator_public_info.Public_key)] = vslices
	s.lk.Unlock()
}
