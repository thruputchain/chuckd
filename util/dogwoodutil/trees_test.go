package dogwoodutil

import (
	"strings"
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func vid(b byte) proto.ValidatorID {
	var id proto.ValidatorID
	id[0] = b
	return id
}

// assertTreeEqual recursively checks two linked trees have identical structure
// and node IDs.
func assertTreeEqual(t *testing.T, want, got *LinkedDAGNode) {
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
		assertTreeEqual(t, want.Children[i], got.Children[i])
	}
}

func TestBuildLinkedTree_Single(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{Node: vid(1)}

	got, err := BuildLinkedTree(m, root.TreeNodes(m))
	if err != nil {
		t.Fatalf("BuildLinkedTree: %v", err)
	}
	assertTreeEqual(t, root, got)
}

func TestBuildLinkedTree_Multilevel(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
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

	got, err := BuildLinkedTree(m, root.TreeNodes(m))
	if err != nil {
		t.Fatalf("BuildLinkedTree: %v", err)
	}
	assertTreeEqual(t, root, got)
}

func TestBuildLinkedTree_DeepLeftSpine(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	// A long left-leaning chain stresses iterative traversal depth.
	const depth = 100
	leaf := &LinkedDAGNode{Node: vid(byte(depth))}
	cur := leaf
	for i := depth - 1; i >= 1; i-- {
		cur = &LinkedDAGNode{
			Node:     vid(byte(i)),
			Children: []*LinkedDAGNode{cur},
		}
	}

	got, err := BuildLinkedTree(m, cur.TreeNodes(m))
	if err != nil {
		t.Fatalf("BuildLinkedTree: %v", err)
	}
	assertTreeEqual(t, cur, got)
}

func TestBuildLinkedTree_PreservesChildOrder(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	// Use distinct sibling ids so a permutation would be detectable.
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(10)},
			{Node: vid(20)},
			{Node: vid(30)},
			{Node: vid(40)},
			{Node: vid(50)},
		},
	}

	got, err := BuildLinkedTree(m, root.TreeNodes(m))
	if err != nil {
		t.Fatalf("BuildLinkedTree: %v", err)
	}
	assertTreeEqual(t, root, got)
	for i, ch := range got.Children {
		if ch.Node != root.Children[i].Node {
			t.Fatalf("child order mismatch at %d: want %x got %x",
				i, root.Children[i].Node, ch.Node)
		}
	}
}

func TestBuildLinkedTree_Empty(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	if _, err := BuildLinkedTree(m, nil); err == nil {
		t.Fatalf("expected error for nil input")
	}
	if _, err := BuildLinkedTree(m, proto.DogwoodTreeNodes{}); err == nil {
		t.Fatalf("expected error for empty input")
	}
}

func TestBuildLinkedTree_DuplicateNode(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2)},
		},
	}
	nodes := root.TreeNodes(m)
	nodes = append(nodes, nodes[1])

	_, err := BuildLinkedTree(m, nodes)
	if err == nil {
		t.Fatalf("expected error for duplicate tree node")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLinkedTree_MissingChild(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2)},
			{Node: vid(3)},
		},
	}
	// TreeNodes is preorder: [root, vid2, vid3]. Drop vid3 so the root's
	// second child reference points to a hash absent from the input.
	nodes := root.TreeNodes(m)
	nodes = nodes[:len(nodes)-1]

	_, err := BuildLinkedTree(m, nodes)
	if err == nil {
		t.Fatalf("expected error for missing child node")
	}
	if !strings.Contains(err.Error(), "missing tree node") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLinkedTree_MissingDeepDescendant(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{
				Node: vid(2),
				Children: []*LinkedDAGNode{
					{Node: vid(3)},
				},
			},
		},
	}
	// Preorder: [root, vid2, vid3]. Drop the deepest node.
	nodes := root.TreeNodes(m)
	nodes = nodes[:len(nodes)-1]

	_, err := BuildLinkedTree(m, nodes)
	if err == nil {
		t.Fatalf("expected error for missing descendant")
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
	assertTreeEqual(t, root, got)
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
