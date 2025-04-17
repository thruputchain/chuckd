package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type DiscoveryService struct {
	cfg    DiscoveryConfig
	dialer libnet.AddressDialer
	//
	privInfo  proto.ValidatorPrivateInfo
	ourSlices []proto.QuorumSlice
	//
	lk        sync.Mutex
	allSlices map[proto.ValidatorID]proto.ValidatorQuorumSlicesSnapshot
	//
	cachedLk        sync.Mutex
	cachedStamp     time.Time
	cachedSnapshots proto.ValidatorQuorumSlicesSnapshots
}

func NewDiscoveryService(
	ctx context.Context,
	cfg DiscoveryConfig,
	dialer libnet.AddressDialer,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
) *DiscoveryService {

	log.Infof("starting discovery service")
	s := &DiscoveryService{
		cfg:       cfg,
		dialer:    dialer,
		privInfo:  privInfo,
		ourSlices: ourSlices,
		allSlices: map[proto.ValidatorID]proto.ValidatorQuorumSlicesSnapshot{},
	}

	go s.advertise(ctx)
	return s
}

func (s *DiscoveryService) Discovery_null(ctx context.Context) error {
	return nil
}

func (s *DiscoveryService) Discovery_update_quorums(
	ctx context.Context,
	vs proto.ValidatorQuorumSlicesSnapshots,
) (proto.UpdateQuorumsResult, error) {
	snapshots := s.CachedSnapshots()
	for _, v := range vs {
		s.addQuorumSlices(v)
	}
	return proto.UpdateQuorumsResult{Quorum_slices_snapshots: snapshots}, nil
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

func (s *DiscoveryService) rebuildSnapshots() proto.ValidatorQuorumSlicesSnapshots {
	s.cachedLk.Lock()
	defer s.cachedLk.Unlock()

	// don't rebuild snapshots more frequently than once every 10 seconds
	if time.Now().Sub(s.cachedStamp) < s.cfg.MinSnapshotTTL {
		return s.cachedSnapshots
	}

	// build snapshots
	s.lk.Lock()
	ourValidatorSlices := protoutil.MakeValidatorQuorumSlicesSnapshot(s.privInfo, s.ourSlices)
	snapshots := proto.ValidatorQuorumSlicesSnapshots{ourValidatorSlices}
	for _, vslices := range s.allSlices {
		snapshots = append(snapshots, vslices)
	}
	protoutil.SortValidatorQuorumSlicesSnapshotsByDescendingTimestamp(snapshots)
	s.lk.Unlock()

	// update cache
	s.cachedSnapshots = snapshots
	s.cachedStamp = time.Now()

	return s.cachedSnapshots
}

func (s *DiscoveryService) CachedSnapshots() proto.ValidatorQuorumSlicesSnapshots {
	return s.rebuildSnapshots()
}
