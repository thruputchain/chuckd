package protoutil

import (
	"sort"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

func MakeValidatorQuorumSlicesSnapshot(
	privInfo proto.ValidatorPrivateInfo,
	slices []proto.QuorumSlice,
) proto.ValidatorQuorumSlicesSnapshot {
	qss := MakeQuorumsSlicesSnapshot(slices)
	return proto.ValidatorQuorumSlicesSnapshot{
		Validator_public_info:  privInfo.Public_info,
		Quorum_slices_snapshot: qss,
		Validator_signature:    SignXdr(privInfo.Secret_key, &qss),
	}
}

func MakeQuorumsSlicesSnapshot(slices []proto.QuorumSlice) proto.QuorumSlicesSnapshot {
	return proto.QuorumSlicesSnapshot{
		Timestamp:     time.Now().UnixNano(),
		Quorum_slices: slices,
	}
}

func VerifyValidatorQuorumSlicesSnapshot(value proto.ValidatorQuorumSlicesSnapshot) bool {
	return VerifyXdr(value.Validator_public_info.Public_key, &value.Quorum_slices_snapshot, value.Validator_signature)
}

func SortValidatorQuorumSlicesSnapshotsByDescendingTimestamp(vslices proto.ValidatorQuorumSlicesSnapshots) {
	ValidatorQuorumSlicesSnapshotsByDescendingTimestamp(vslices).Sort()
}

type ValidatorQuorumSlicesSnapshotsByDescendingTimestamp proto.ValidatorQuorumSlicesSnapshots

func (xs ValidatorQuorumSlicesSnapshotsByDescendingTimestamp) Len() int { return len(xs) }
func (xs ValidatorQuorumSlicesSnapshotsByDescendingTimestamp) Less(i, j int) bool {
	return xs[i].Quorum_slices_snapshot.Timestamp > xs[j].Quorum_slices_snapshot.Timestamp
}
func (xs ValidatorQuorumSlicesSnapshotsByDescendingTimestamp) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}
func (xs ValidatorQuorumSlicesSnapshotsByDescendingTimestamp) Sort() { sort.Sort(xs) }

// ValidatorsInQuorumSlices returns the validators encountered in quorum slices in random order.
func ValidatorsInQuorumSlices(slices []proto.QuorumSlice) []proto.ValidatorPublicInfo {
	m := map[proto.ValidatorID]proto.ValidatorPublicInfo{}
	for _, s := range slices {
		for _, v := range s {
			m[ValidatorPublicKeyToID(v.Public_key)] = v
		}
	}
	vs := []proto.ValidatorPublicInfo{}
	for _, v := range m {
		vs = append(vs, v)
	}
	return vs
}
