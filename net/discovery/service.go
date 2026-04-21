package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type Discovery interface {
	SelfID() proto.ValidatorID
	OurQuorumSet() proto.SCPQuorumSet
	SecretKey() proto.ValidatorSecretKey
	LookupValidator(vid proto.ValidatorID) (proto.ValidatorPublicInfo, bool)
	LookupTrustedValidator(vid proto.ValidatorID) (proto.ValidatorPublicInfo, bool)
	CachedSnapshots() proto.KnownQuorumSets
	GetQuorumSetLocally(ctx context.Context, hash proto.SCPHash) (*proto.SCPQuorumSet, error)
}

type DiscoveryService struct {
	cfg proto.DiscoveryConfig
	net libnet.Network
	//
	privInfo      proto.ValidatorPrivateInfo
	ourQSet       proto.SCPQuorumSet
	ourValidators []proto.ValidatorPublicInfo
	ourVIDs       map[proto.ValidatorID]proto.ValidatorPublicInfo
	//
	lk         sync.Mutex
	knownQSets map[proto.ValidatorID]proto.SignedQuorumSetSnapshot // validator_id -> their own qset
	hashQSet   map[proto.SCPHash]proto.SCPQuorumSet                // hash of qset -> qset
	knownVs    map[proto.ValidatorID]proto.SignedValidatorInfoSnapshot
	//
	scpController scpkit.SCPKnownQSets
	//
	cachedLk         sync.Mutex
	cachedStamp      time.Time
	cachedSnapshots  []proto.SignedQuorumSetSnapshot
	cachedValidators []proto.SignedValidatorInfoSnapshot
	//
	m libxdr.XdrMarshaler
	//
	stopOnce sync.Once
	done     chan struct{}
}

func NewDiscoveryService(
	ctx context.Context,
	n libnet.Network,
	cfg proto.DiscoveryConfig,
	vcfg proto.ValidatorConfig,
	scfg proto.SCPConfig,
) *DiscoveryService {

	scpkit.NormalizeQSet(&scfg.Quorum_set, nil)

	s := &DiscoveryService{
		cfg:           cfg,
		net:           n,
		privInfo:      vcfg.Private_info,
		ourQSet:       scfg.Quorum_set,
		ourValidators: vcfg.Known_validators,
		knownQSets:    map[proto.ValidatorID]proto.SignedQuorumSetSnapshot{},
		hashQSet:      map[proto.SCPHash]proto.SCPQuorumSet{},
		knownVs:       map[proto.ValidatorID]proto.SignedValidatorInfoSnapshot{},
		m:             libxdr.NewXdrMarshaler(),
		done:          make(chan struct{}),
	}

	s.ourVIDs = map[proto.ValidatorID]proto.ValidatorPublicInfo{}
	s.ourVIDs[s.SelfID()] = s.Self()
	for _, v := range s.ourValidators {
		s.ourVIDs[protoutil.PublicVID(v)] = v
	}

	return s
}

func (s *DiscoveryService) Start(ctx context.Context, controller scpkit.SCPKnownQSets) {
	liblog.Infof(ctx, "starting discovery service")

	s.scpController = controller

	s.lk.Lock()
	s.setQuorumSet(ctx, s.ourQSet)
	s.lk.Unlock()

	go s.advertise(ctx)
}

func (s *DiscoveryService) Stop() {
	s.stopOnce.Do(func() {
		close(s.done)
	})
}

func (s *DiscoveryService) IsStopped() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

func (s *DiscoveryService) OurQuorumSet() proto.SCPQuorumSet {
	return s.ourQSet
}

func (s *DiscoveryService) setQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) {
	s.scpController.SetKnownQuorumSet(ctx, qset)
	s.hashQSet[libxdr.HashXdr(s.m, &qset)] = qset
}

// XXX: qset garbage collection not implemented
func (s *DiscoveryService) removeQuorumSet(ctx context.Context, qhash proto.SCPHash) {
	_, ok := s.hashQSet[qhash]
	delete(s.hashQSet, qhash)
	if ok {
		s.scpController.RemoveKnownQuorumSet(ctx, qhash)
	}
}

func (s *DiscoveryService) GetQuorumSetLocally(ctx context.Context, hash proto.SCPHash) (*proto.SCPQuorumSet, error) {
	s.lk.Lock()
	defer s.lk.Unlock()

	qset, ok := s.hashQSet[hash]
	if ok {
		return &qset, nil
	} else {
		return nil, nil
	}
}

func (s *DiscoveryService) SecretKey() proto.ValidatorSecretKey {
	return s.privInfo.Secret_key
}

func (s *DiscoveryService) Self() proto.ValidatorPublicInfo {
	return s.privInfo.Public_info
}

func (s *DiscoveryService) SelfID() proto.ValidatorID {
	return protoutil.PublicVID(s.Self())
}

func (s *DiscoveryService) LookupValidator(vid proto.ValidatorID) (proto.ValidatorPublicInfo, bool) {

	pub, ok := s.ourVIDs[vid]
	if ok {
		return pub, true
	}

	s.lk.Lock()
	spub, ok := s.knownVs[vid]
	s.lk.Unlock()
	if ok {
		return spub.Validator_snapshot.Validator, true
	}

	return proto.ValidatorPublicInfo{}, false
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
	s.updateKnown(ctx, known)
	return cachedKnown, nil
}

func (s *DiscoveryService) updateKnown(ctx context.Context, known proto.KnownQuorumSets) {
	s.lk.Lock()
	defer s.lk.Unlock()

	for _, qs := range known.Quorum_set_snapshots {
		s.addQuorumSet(ctx, qs)
	}
	for _, vs := range known.Validator_snapshots {
		s.addValidator(vs)
	}
}

func (s *DiscoveryService) addQuorumSet(ctx context.Context, qset proto.SignedQuorumSetSnapshot) {
	if !protoutil.VerifySignedQuorumSetSnapshot(s.m, qset) {
		return
	}

	// add qsets only if more recent than currently known
	key := protoutil.VPublicKeyToID(qset.Signer.Public_key)
	if current, ok := s.knownQSets[key]; ok {
		if current.Quorum_set_snapshot.Timestamp < qset.Quorum_set_snapshot.Timestamp {
			s.knownQSets[key] = qset
		}
	} else {
		s.knownQSets[key] = qset
	}

	// index by qset hash
	scpkit.NormalizeQSet(&qset.Quorum_set_snapshot.Quorum_set, nil)
	s.setQuorumSet(ctx, qset.Quorum_set_snapshot.Quorum_set)
}

func (s *DiscoveryService) addValidator(v proto.SignedValidatorInfoSnapshot) {

	//XXX: determine if the signer is trusted

	if !protoutil.VerifySignedValidatorInfoSnapshot(s.m, v) {
		return
	}

	// add validator info only if more recent than currently known
	key := protoutil.VPublicKeyToID(v.Validator_snapshot.Validator.Public_key)
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
	if time.Since(s.cachedStamp) < time.Duration(s.cfg.Snapshot_ttl_msec)*time.Millisecond {
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
	for _, v := range s.ourValidators {
		vv[protoutil.PublicVID(v)] = protoutil.MakeSignedValidatorInfoSnapshot(s.m, s.privInfo, v)
	}
	for _, v := range s.knownVs {
		key := protoutil.PublicVID(v.Signer)
		_, ok := vv[key]
		if !ok { // prioritize our info
			vv[key] = v
		}
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
