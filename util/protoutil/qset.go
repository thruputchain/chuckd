package protoutil

import (
	"fmt"

	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func VerifyQSet(qset proto.SCPQuorumSet) error {
	if !VerifyQSetDepth(qset, systemfixed.MaxQuorumSetNestingDepth) {
		return fmt.Errorf("quorum set nesting depth is more than %d", systemfixed.MaxQuorumSetNestingDepth)
	}
	// Check all union tags before marshaling or normalizing network quorum sets.
	if err := verifyQSetPublicKeys(qset); err != nil {
		return err
	}
	if !VerifyQSetSize(qset, int(proto.KCoordinatorMaxBytesSCPQuorumSet)) {
		return fmt.Errorf("quorum set size is more than %d", proto.KCoordinatorMaxBytesSCPQuorumSet)
	}
	return nil
}

func verifyQSetPublicKeys(qset proto.SCPQuorumSet) error {
	for _, key := range qset.Validators {
		if key.Scheme != proto.Ed25519 {
			return fmt.Errorf("unsupported public key scheme %d in quorum set", key.Scheme)
		}
	}
	for _, inner := range qset.Inner_sets {
		if err := verifyQSetPublicKeys(inner); err != nil {
			return err
		}
	}
	return nil
}

type qsetDepthFrame struct {
	qset  proto.SCPQuorumSet
	depth uint32
}

// VerifyQSetSize returns true when the encoded size of qset is <= maxSize.
func VerifyQSetSize(qset proto.SCPQuorumSet, maxSize int) bool {
	if maxSize < 0 {
		return false
	}
	m := libxdr.NewXdrMarshaler()
	bb := m.MarshalXdr(&qset)
	defer bb.Recycle()
	return len(bb.Bytes()) <= maxSize
}

// VerifyQSetDepth returns true when the maximum nesting depth of qset is <= maxDepth.
// The root quorum set has depth 0.
func VerifyQSetDepth(qset proto.SCPQuorumSet, maxDepth uint32) bool {
	stack := []qsetDepthFrame{{qset: qset, depth: 0}}
	for len(stack) > 0 {
		last := len(stack) - 1
		frame := stack[last]
		stack = stack[:last]

		if frame.depth > maxDepth {
			return false
		}

		nextDepth := frame.depth + 1
		for _, inner := range frame.qset.Inner_sets {
			stack = append(stack, qsetDepthFrame{
				qset:  inner,
				depth: nextDepth,
			})
		}
	}
	return true
}

// VerifyQuorumSetSnapshotDepth applies VerifyQSetDepth to the snapshot's root quorum set.
func VerifyQuorumSetSnapshotDepth(snapshot proto.QuorumSetSnapshot, maxDepth uint32) bool {
	return VerifyQSetDepth(snapshot.Quorum_set, maxDepth)
}

// VerifyKnownQuorumSetsDepth verifies depth for every quorum set snapshot in known.
func VerifyKnownQuorumSetsDepth(known proto.KnownQuorumSets, maxDepth uint32) bool {
	for _, snapshot := range known.Quorum_set_snapshots {
		if !VerifyQuorumSetSnapshotDepth(snapshot.Quorum_set_snapshot, maxDepth) {
			return false
		}
	}
	return true
}
