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

	go s.advertise(ctx)
	return s
}

func (s *DiscoveryService) Discovery_null() {}

func (s *DiscoveryService) Discovery_update_quorums(vs proto.ValidatorQuorumSlicesSnapshots) proto.UpdateQuorumsResult {
	snapshots := s.Snapshots()
	for _, v := range vs {
		s.addQuorumSlices(v)
	}
	return proto.UpdateQuorumsResult{Quorum_slices_snapshots: snapshots}
}

func (s *DiscoveryService) addQuorumSlices(vslices proto.ValidatorQuorumSlicesSnapshot) {
	if !protoutil.VerifyValidatorQuorumSlicesSnapshot(vslices) {
		return
	}
	s.lk.Lock()
	// add slices only if more recent than currently known
	key := protoutil.ValidatorPublicKeyToID(vslices.Validator_public_info.Public_key)
	if current, ok := s.allSlices[key]; ok {
		if current.Quorum_slices_snapshot.Timestamp < vslices.Quorum_slices_snapshot.Timestamp {
			s.allSlices[key] = vslices
		}
	} else {
		s.allSlices[key] = vslices
	}
	s.lk.Unlock()
}

func (s *DiscoveryService) Snapshots() proto.ValidatorQuorumSlicesSnapshots {
	ourValidatorSlices := protoutil.MakeValidatorQuorumSlicesSnapshot(s.privInfo, s.ourSlices)
	snapshots := proto.ValidatorQuorumSlicesSnapshots{ourValidatorSlices}

	s.lk.Lock()
	for _, vslices := range s.allSlices {
		snapshots = append(snapshots, vslices)
	}
	s.lk.Unlock()

	protoutil.SortValidatorQuorumSlicesSnapshotsByDescendingTimestamp(snapshots)
	return snapshots
}
