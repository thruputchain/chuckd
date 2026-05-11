package discovery

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (s *DiscoveryService) snapshot(ctx context.Context) {
	interval := time.Duration(s.cfg.Snapshot_ttl_msec) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RebuildSnapshot(ctx)
		}
	}
}

func (s *DiscoveryService) CachedSnapshots() proto.KnownQuorumSets {
	s.cachedLk.Lock()
	defer s.cachedLk.Unlock()
	return proto.KnownQuorumSets{
		Quorum_set_snapshots: s.cachedSnapshots,
		Validator_snapshots:  s.cachedValidators,
	}
}

func (s *DiscoveryService) RebuildSnapshot(ctx context.Context) {

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
	s.cachedLk.Lock()
	defer s.cachedLk.Unlock()

	s.cachedSnapshots = allQsets
	s.cachedValidators = allVs
	s.cachedStamp = time.Now()
}
