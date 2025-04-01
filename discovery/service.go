package discovery

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// TODO: interface with multicast trees and broadcast

// TODO: start the discovery service on "server start"
// TODO: share our quorums, when?

type DiscoveryService struct {
	slices  []proto.QuorumSlice
	lk      sync.Mutex
	vslices map[proto.ValidatorID]proto.ValidatorQuorumSlicesSnapshot
}

func NewDiscoveryService(
	ctx context.Context,
	slices []proto.QuorumSlice,
) *DiscoveryService {
	s := &DiscoveryService{
		slices:  slices,
		vslices: map[proto.ValidatorID]proto.ValidatorQuorumSlicesSnapshot{},
	}
	go s.advertise(ctx)
	return s
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
