package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type DiscoveryService struct {
	cfg proto.DiscoveryConfig
	net libnet.Network
	//
	privInfo      proto.ValidatorPrivateInfo
	ourQSet       proto.QuorumSet
	ourValidators []proto.ValidatorPublicInfo
	ourVIDs       map[proto.ValidatorID]proto.ValidatorPublicInfo
	//
	lk         sync.Mutex
	knownQSets map[proto.ValidatorID]proto.SignedQuorumSetSnapshot
	knownVs    map[proto.ValidatorID]proto.SignedValidatorInfoSnapshot
	//
	cachedLk         sync.Mutex
	cachedStamp      time.Time
	cachedSnapshots  []proto.SignedQuorumSetSnapshot
	cachedValidators []proto.SignedValidatorInfoSnapshot
	//
	m xdrutil.XdrMarshaler
}

func NewDiscoveryService(
	ctx context.Context,
	cfg proto.DiscoveryConfig,
	n libnet.Network,
	vcfg proto.ValidatorConfig,
) *DiscoveryService {

	liblog.Infof("starting discovery service")
	s := &DiscoveryService{
		cfg:           cfg,
		net:           n,
		privInfo:      vcfg.Private_info,
		ourQSet:       vcfg.Quorum_set,
		ourValidators: vcfg.Known_validators,
		knownQSets:    map[proto.ValidatorID]proto.SignedQuorumSetSnapshot{},
		knownVs:       map[proto.ValidatorID]proto.SignedValidatorInfoSnapshot{},
		m:             xdrutil.NewXdrMarshaler(),
	}

	s.ourVIDs = map[proto.ValidatorID]proto.ValidatorPublicInfo{}
	s.ourVIDs[s.SelfID()] = s.Self()
	for _, v := range s.ourValidators {
		s.ourVIDs[protoutil.VIDOfPublic(v)] = v
	}

	go s.advertise(ctx)
	return s
}

func (s *DiscoveryService) Self() proto.ValidatorPublicInfo {
	return s.privInfo.Public_info
}

func (s *DiscoveryService) SelfID() proto.ValidatorID {
	return protoutil.VIDOfPublic(s.Self())
}

func (s *DiscoveryService) LookupTrustedValidator(vid proto.ValidatorID) (proto.ValidatorPublicInfo, bool) {
	v, ok := s.ourVIDs[vid]
	return v, ok
}

func (s *DiscoveryService) Discovery_null(ctx context.Context) error {
	return nil
}

func (s *DiscoveryService) Discovery_update_quorums(
	ctx context.Context,
	known proto.KnownQuorumSets,
) (proto.KnownQuorumSets, error) {

	cachedKnown := s.CachedSnapshots()
	s.updateKnown(known)
	return cachedKnown, nil
}

func (s *DiscoveryService) updateKnown(known proto.KnownQuorumSets) {
	s.lk.Lock()
	defer s.lk.Unlock()

	for _, qs := range known.Quorum_set_snapshots {
		s.addQuorumSet(qs)
	}
	for _, vs := range known.Validator_snapshots {
		s.addValidator(vs)
	}
}

func (s *DiscoveryService) addQuorumSet(qset proto.SignedQuorumSetSnapshot) {
	if !protoutil.VerifySignedQuorumSetSnapshot(s.m, qset) {
		return
	}

	// add qsets only if more recent than currently known
	key := protoutil.ValidatorPublicKeyToID(qset.Signer.Public_key)
	if current, ok := s.knownQSets[key]; ok {
		if current.Quorum_set_snapshot.Timestamp < qset.Quorum_set_snapshot.Timestamp {
			s.knownQSets[key] = qset
		}
	} else {
		s.knownQSets[key] = qset
	}
}

func (s *DiscoveryService) addValidator(v proto.SignedValidatorInfoSnapshot) {
	if !protoutil.VerifySignedValidatorInfoSnapshot(s.m, v) {
		return
	}

	// add qsets only if more recent than currently known
	key := protoutil.ValidatorPublicKeyToID(v.Signer.Public_key)
	if current, ok := s.knownVs[key]; ok {
		if current.Validator_snapshot.Timestamp < v.Validator_snapshot.Timestamp {
			s.knownVs[key] = v
		}
	} else {
		s.knownVs[key] = v
	}
}

func (s *DiscoveryService) CachedSnapshots() proto.KnownQuorumSets {
	s.cachedLk.Lock()
	defer s.cachedLk.Unlock()

	// don't rebuild snapshots more frequently than once every 10 seconds
	if time.Now().Sub(s.cachedStamp) < time.Duration(s.cfg.Snapshot_ttl_msec)*time.Millisecond {
		return proto.KnownQuorumSets{
			Quorum_set_snapshots: s.cachedSnapshots,
			Validator_snapshots:  s.cachedValidators,
		}
	}

	// build snapshots
	s.lk.Lock()

	// qsets
	ourSignedQSet := protoutil.MakeSignedQuorumSetSnapshot(s.m, s.privInfo, s.ourQSet)
	allQsets := []proto.SignedQuorumSetSnapshot{ourSignedQSet}
	for _, signedQSet := range s.knownQSets {
		allQsets = append(allQsets, signedQSet)
	}
	protoutil.SortSignedQuorumSetSnapshotsByDescendingTimestamp(allQsets)

	// validators
	vv := map[proto.ValidatorID]proto.SignedValidatorInfoSnapshot{}
	for _, v := range s.knownVs {
		vv[protoutil.VIDOfPublic(v.Signer)] = v
	}
	for _, v := range s.ourValidators {
		vv[protoutil.VIDOfPublic(v)] = protoutil.MakeSignedValidatorInfoSnapshot(s.m, s.privInfo, v)
	}
	allVs := []proto.SignedValidatorInfoSnapshot{}
	for _, v := range vv {
		allVs = append(allVs, v)
	}
	protoutil.SortSignedValidatorInfoSnapshots(allVs)

	s.lk.Unlock()

	// update cache
	s.cachedSnapshots = allQsets
	s.cachedValidators = allVs
	s.cachedStamp = time.Now()

	return proto.KnownQuorumSets{
		Quorum_set_snapshots: s.cachedSnapshots,
		Validator_snapshots:  allVs,
	}
}
