package protoutil

import (
	"sort"
	"time"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func IsNodeInQuorumSet(node proto.SCPNodeID, qset proto.SCPQuorumSet) bool {
	q := NodeToValidatorID(node)
	for _, n := range qset.Validators {
		if NodeToValidatorID(n) == q {
			return true
		}
	}
	for _, inner := range qset.Inner_sets {
		if IsNodeInQuorumSet(node, inner) {
			return true
		}
	}
	return false
}

func MakeSignedQuorumSetSnapshot(
	m libxdr.XdrMarshaler,
	privInfo proto.ValidatorPrivateInfo,
	qset proto.SCPQuorumSet,
) proto.SignedQuorumSetSnapshot {
	qss := proto.QuorumSetSnapshot{
		Timestamp:  TimepointFromTime(time.Now()),
		Quorum_set: qset,
	}
	return proto.SignedQuorumSetSnapshot{
		Signer:              privInfo.Public_info,
		Signature:           SignXdr(m, privInfo.Secret_key, &qss),
		Quorum_set_snapshot: qss,
	}
}

func MakeSignedValidatorInfoSnapshot(
	m libxdr.XdrMarshaler,
	privInfo proto.ValidatorPrivateInfo,
	v proto.ValidatorPublicInfo,
) proto.SignedValidatorInfoSnapshot {
	w := proto.ValidatorInfoSnapshot{
		Timestamp: TimepointFromTime(time.Now()),
		Validator: v,
	}
	return proto.SignedValidatorInfoSnapshot{
		Signer:             privInfo.Public_info,
		Signature:          SignXdr(m, privInfo.Secret_key, &w),
		Validator_snapshot: w,
	}
}

// sorting qsets

func SortSignedQuorumSetSnapshotsByDescendingTimestamp(ss []proto.SignedQuorumSetSnapshot) {
	SignedQuorumSetSnapshotsByDescendingTimestamp(ss).Sort()
}

type SignedQuorumSetSnapshotsByDescendingTimestamp []proto.SignedQuorumSetSnapshot

func (xs SignedQuorumSetSnapshotsByDescendingTimestamp) Len() int {
	return len(xs)
}

func (xs SignedQuorumSetSnapshotsByDescendingTimestamp) Less(i, j int) bool {
	return xs[i].Quorum_set_snapshot.Timestamp > xs[j].Quorum_set_snapshot.Timestamp
}

func (xs SignedQuorumSetSnapshotsByDescendingTimestamp) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

func (xs SignedQuorumSetSnapshotsByDescendingTimestamp) Sort() { sort.Sort(xs) }

// sorting validators

func SortSignedValidatorInfoSnapshots(vs []proto.SignedValidatorInfoSnapshot) {
	SignedValidatorInfoSnapshotsByDescendingTimestamp(vs).Sort()
}

type SignedValidatorInfoSnapshotsByDescendingTimestamp []proto.SignedValidatorInfoSnapshot

func (xs SignedValidatorInfoSnapshotsByDescendingTimestamp) Len() int {
	return len(xs)
}

func (xs SignedValidatorInfoSnapshotsByDescendingTimestamp) Less(i, j int) bool {
	return xs[i].Validator_snapshot.Timestamp > xs[j].Validator_snapshot.Timestamp
}

func (xs SignedValidatorInfoSnapshotsByDescendingTimestamp) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

func (xs SignedValidatorInfoSnapshotsByDescendingTimestamp) Sort() { sort.Sort(xs) }

// verification

func VerifySignedQuorumSetSnapshot(m libxdr.XdrMarshaler, v proto.SignedQuorumSetSnapshot) bool {
	if err := VerifyQSet(v.Quorum_set_snapshot.Quorum_set); err != nil {
		return false
	}
	return VerifyXdr(m, v.Signer.Public_key, &v.Quorum_set_snapshot, v.Signature)
}

func VerifySignedValidatorInfoSnapshot(m libxdr.XdrMarshaler, v proto.SignedValidatorInfoSnapshot) bool {
	if v.Validator_snapshot.Validator.Public_key.Scheme != proto.Ed25519 {
		return false
	}
	return VerifyXdr(m, v.Signer.Public_key, &v.Validator_snapshot, v.Signature)
}

// misc

func SnapshotValidators(known proto.KnownQuorumSets) []proto.ValidatorPublicInfo {
	vs := []proto.ValidatorPublicInfo{}
	for _, v := range known.Validator_snapshots {
		vs = append(vs, v.Validator_snapshot.Validator)
	}
	return vs
}

func SnapshotValidatorIDs(known proto.KnownQuorumSets) map[proto.ValidatorID]bool {
	vs := map[proto.ValidatorID]bool{}
	for _, v := range known.Validator_snapshots {
		vs[PublicVID(v.Validator_snapshot.Validator)] = true
	}
	return vs
}
