package dogwoodutil

import (
	"testing"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func vid(b byte) proto.ValidatorID {
	var id proto.ValidatorID
	id[0] = b
	return id
}

// assertNodesEqual recursively checks two linked trees have identical structure
// and node IDs.
func assertNodesEqual(t *testing.T, want, got *LinkedDAGNode) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if want == nil || got == nil {
		t.Fatalf("nil mismatch: want=%v got=%v", want, got)
	}
	if want.Node != got.Node {
		t.Fatalf("node mismatch: want %x got %x", want.Node, got.Node)
	}
	if len(want.Children) != len(got.Children) {
		t.Fatalf("children len mismatch at %x: want %d got %d",
			want.Node, len(want.Children), len(got.Children))
	}
	for i := range want.Children {
		assertNodesEqual(t, want.Children[i], got.Children[i])
	}
}

func TestReconstructLinkedTree_RoundTrip(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2)},
			{
				Node: vid(3),
				Children: []*LinkedDAGNode{
					{Node: vid(4)},
				},
			},
		},
	}
	hash := TreeHash(m, root)

	got, err := ReconstructLinkedTree(m, hash, root.TreeNodes(m))
	if err != nil {
		t.Fatalf("ReconstructLinkedTree: %v", err)
	}
	assertNodesEqual(t, root, got)
	if h := TreeHash(m, got); h != hash {
		t.Fatalf("rebuilt tree hash mismatch: want %x got %x", hash, h)
	}
}

func TestReconstructLinkedTree_HashMismatch(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2)},
		},
	}
	var wrong proto.DogwoodTreeHash // zero value, will not match any real hash
	_, err := ReconstructLinkedTree(m, wrong, root.TreeNodes(m))
	if err == nil {
		t.Fatalf("expected error for root hash mismatch")
	}
}

func TestReconstructLinkedTree_Empty(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	var h proto.DogwoodTreeHash
	if _, err := ReconstructLinkedTree(m, h, nil); err == nil {
		t.Fatalf("expected error for empty input")
	}
}

func TestIsTree_Nil(t *testing.T) {
	var l *LinkedDAGNode
	if !l.IsTree() {
		t.Fatalf("nil receiver should be a tree")
	}
}

func TestIsTree_Single(t *testing.T) {
	root := &LinkedDAGNode{Node: vid(1)}
	if !root.IsTree() {
		t.Fatalf("single node should be a tree")
	}
}

func TestIsTree_Tree(t *testing.T) {
	// Tree:
	//        1
	//      / | \
	//     2  3  4
	//    /|     |
	//   5 6     7
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{
				Node: vid(2),
				Children: []*LinkedDAGNode{
					{Node: vid(5)},
					{Node: vid(6)},
				},
			},
			{Node: vid(3)},
			{
				Node: vid(4),
				Children: []*LinkedDAGNode{
					{Node: vid(7)},
				},
			},
		},
	}
	if !root.IsTree() {
		t.Fatalf("multilevel tree should be a tree")
	}
}

func TestIsTree_SharedLeaf(t *testing.T) {
	// Diamond: two parents share the same leaf pointer.
	//      1
	//     / \
	//    2   3
	//     \ /
	//      4
	shared := &LinkedDAGNode{Node: vid(4)}
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2), Children: []*LinkedDAGNode{shared}},
			{Node: vid(3), Children: []*LinkedDAGNode{shared}},
		},
	}
	if root.IsTree() {
		t.Fatalf("DAG with shared leaf should not be a tree")
	}
}

func TestIsTree_SharedSubtree(t *testing.T) {
	// Two parents share an entire subtree (sub) by pointer.
	sub := &LinkedDAGNode{
		Node: vid(10),
		Children: []*LinkedDAGNode{
			{Node: vid(11)},
			{Node: vid(12)},
		},
	}
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2), Children: []*LinkedDAGNode{sub}},
			{Node: vid(3), Children: []*LinkedDAGNode{sub}},
		},
	}
	if root.IsTree() {
		t.Fatalf("DAG with shared subtree should not be a tree")
	}
}

func TestIsTree_DistinctEqualSubtrees(t *testing.T) {
	// Two parents reference structurally identical but distinct subtree
	// instances. Since pointers differ, this is still a tree.
	mkSub := func() *LinkedDAGNode {
		return &LinkedDAGNode{
			Node: vid(10),
			Children: []*LinkedDAGNode{
				{Node: vid(11)},
			},
		}
	}
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2), Children: []*LinkedDAGNode{mkSub()}},
			{Node: vid(3), Children: []*LinkedDAGNode{mkSub()}},
		},
	}
	if !root.IsTree() {
		t.Fatalf("distinct equal subtrees should still form a tree")
	}
}

func TestIsTree_BuildLinkedDAGSharesIdenticalSubtrees(t *testing.T) {
	// BuildLinkedDAG de-duplicates identical subtrees into shared pointers,
	// so the result should not be a tree when duplicate subtrees exist.
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{
				Node: vid(2),
				Children: []*LinkedDAGNode{
					{Node: vid(10)},
				},
			},
			{
				Node: vid(3),
				Children: []*LinkedDAGNode{
					{Node: vid(10)},
				},
			},
		},
	}

	dag, err := BuildLinkedDAG(m, root.TreeNodes(m))
	if err != nil {
		t.Fatalf("BuildLinkedDAG: %v", err)
	}
	if dag.IsTree() {
		t.Fatalf("BuildLinkedDAG output with duplicate subtrees should not be a tree")
	}
}
