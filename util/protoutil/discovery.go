package protoutil

import (
	"sort"
	"time"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func MakeSignedQuorumSetSnapshot(
	m xdrutil.XdrMarshaler,
	privInfo proto.ValidatorPrivateInfo,
	qset proto.SCPQuorumSet,
) proto.SignedQuorumSetSnapshot {
	qss := proto.QuorumSetSnapshot{
		Timestamp:  uint64(time.Now().UnixNano()),
		Quorum_set: qset,
	}
	return proto.SignedQuorumSetSnapshot{
		Signer:              privInfo.Public_info,
		Signature:           SignXdr(m, privInfo.Secret_key, &qss),
		Quorum_set_snapshot: qss,
	}
}

func MakeSignedValidatorInfoSnapshot(
	m xdrutil.XdrMarshaler,
	privInfo proto.ValidatorPrivateInfo,
	v proto.ValidatorPublicInfo,
) proto.SignedValidatorInfoSnapshot {
	w := proto.ValidatorInfoSnapshot{
		Timestamp: uint64(time.Now().UnixNano()),
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

func VerifySignedQuorumSetSnapshot(m xdrutil.XdrMarshaler, v proto.SignedQuorumSetSnapshot) bool {
	return VerifyXdr(m, v.Signer.Public_key, &v.Quorum_set_snapshot, v.Signature)
}

func VerifySignedValidatorInfoSnapshot(m xdrutil.XdrMarshaler, v proto.SignedValidatorInfoSnapshot) bool {
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
