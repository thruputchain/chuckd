package protoutil

import (
	"testing"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func TestVerifyQSetSize(t *testing.T) {
	small := proto.SCPQuorumSet{Threshold: 1}
	smallSize := qsetEncodedSize(small)

	if !VerifyQSetSize(small, smallSize) {
		t.Fatalf("expected quorum set to pass at exact encoded size %d", smallSize)
	}
	if VerifyQSetSize(small, smallSize-1) {
		t.Fatalf("expected quorum set to fail when maxSize is one byte too small")
	}
	if VerifyQSetSize(small, -1) {
		t.Fatalf("expected negative maxSize to fail")
	}

	large := makeNestedQSetChain(3)
	largeSize := qsetEncodedSize(large)
	if largeSize <= smallSize {
		t.Fatalf("expected nested quorum set size %d to exceed base size %d", largeSize, smallSize)
	}
	if !VerifyQSetSize(large, largeSize) {
		t.Fatalf("expected nested quorum set to pass at exact encoded size %d", largeSize)
	}
	if VerifyQSetSize(large, smallSize) {
		t.Fatalf("expected nested quorum set to fail maxSize=%d", smallSize)
	}
}

func TestVerifyQSetDepth(t *testing.T) {
	depth0 := proto.SCPQuorumSet{
		Threshold: 1,
	}
	if !VerifyQSetDepth(depth0, 0) {
		t.Fatalf("expected depth-0 quorum set to pass maxDepth=0")
	}

	depth1 := proto.SCPQuorumSet{
		Threshold: 1,
		Inner_sets: []proto.SCPQuorumSet{
			{Threshold: 1},
		},
	}
	if VerifyQSetDepth(depth1, 0) {
		t.Fatalf("expected depth-1 quorum set to fail maxDepth=0")
	}
	if !VerifyQSetDepth(depth1, 1) {
		t.Fatalf("expected depth-1 quorum set to pass maxDepth=1")
	}
}

func TestVerifyQSetDepthDeepChain(t *testing.T) {
	const chainDepth = 5000
	qset := makeNestedQSetChain(chainDepth)

	if !VerifyQSetDepth(qset, chainDepth) {
		t.Fatalf("expected chain depth %d to pass maxDepth=%d", chainDepth, chainDepth)
	}
	if VerifyQSetDepth(qset, chainDepth-1) {
		t.Fatalf("expected chain depth %d to fail maxDepth=%d", chainDepth, chainDepth-1)
	}
}

func TestVerifyKnownQuorumSetsDepth(t *testing.T) {
	known := proto.KnownQuorumSets{
		Quorum_set_snapshots: []proto.SignedQuorumSetSnapshot{
			{
				Quorum_set_snapshot: proto.QuorumSetSnapshot{
					Quorum_set: makeNestedQSetChain(2),
				},
			},
			{
				Quorum_set_snapshot: proto.QuorumSetSnapshot{
					Quorum_set: makeNestedQSetChain(4),
				},
			},
		},
	}

	if VerifyKnownQuorumSetsDepth(known, 3) {
		t.Fatalf("expected known quorum sets to fail when one snapshot exceeds depth limit")
	}
	if !VerifyKnownQuorumSetsDepth(known, 4) {
		t.Fatalf("expected known quorum sets to pass when all snapshots are within depth limit")
	}
}

func makeNestedQSetChain(depth uint32) proto.SCPQuorumSet {
	root := proto.SCPQuorumSet{Threshold: 1}
	current := &root
	for range depth {
		current.Inner_sets = []proto.SCPQuorumSet{
			{Threshold: 1},
		}
		current = &current.Inner_sets[0]
	}
	return root
}

func qsetEncodedSize(qset proto.SCPQuorumSet) int {
	m := libxdr.NewXdrMarshaler()
	bb := m.MarshalXdr(&qset)
	defer bb.Recycle()
	return len(bb.Bytes())
}
