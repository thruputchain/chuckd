package dogwoodutil

import (
	"testing"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// assertDAGMatchesTree asserts that BuildLinkedDAG produces a structure
// matching the given tree.
func assertDAGMatchesTree(t *testing.T, root *LinkedDAGNode) {
	t.Helper()
	m := libxdr.NewXdrMarshaler()
	nodes := root.TreeNodes(m)

	gotDAG, err := BuildLinkedDAG(m, nodes)
	if err != nil {
		t.Fatalf("BuildLinkedDAG: %v", err)
	}
	assertNodesEqual(t, root, gotDAG)
}

func TestBuildLinkedDAG_AgreesWithTree_Single(t *testing.T) {
	assertDAGMatchesTree(t, &LinkedDAGNode{Node: vid(1)})
}

func TestBuildLinkedDAG_AgreesWithTree_Multilevel(t *testing.T) {
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
	assertDAGMatchesTree(t, root)
}

func TestBuildLinkedDAG_AgreesWithTree_DeepLeftSpine(t *testing.T) {
	const depth = 100
	leaf := &LinkedDAGNode{Node: vid(byte(depth))}
	cur := leaf
	for i := depth - 1; i >= 1; i-- {
		cur = &LinkedDAGNode{
			Node:     vid(byte(i)),
			Children: []*LinkedDAGNode{cur},
		}
	}
	assertDAGMatchesTree(t, cur)
}

func TestBuildLinkedDAG_AgreesWithTree_PreservesChildOrder(t *testing.T) {
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

	m := libxdr.NewXdrMarshaler()
	got, err := BuildLinkedDAG(m, root.TreeNodes(m))
	if err != nil {
		t.Fatalf("BuildLinkedDAG: %v", err)
	}
	assertNodesEqual(t, root, got)
	for i, ch := range got.Children {
		if ch.Node != root.Children[i].Node {
			t.Fatalf("child order mismatch at %d: want %x got %x",
				i, root.Children[i].Node, ch.Node)
		}
	}
}

func TestBuildLinkedDAG_Empty(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	if _, err := BuildLinkedDAG(m, nil); err == nil {
		t.Fatalf("expected error for nil input")
	}
	if _, err := BuildLinkedDAG(m, proto.DogwoodTreeNodes{}); err == nil {
		t.Fatalf("expected error for empty input")
	}
}

func TestBuildLinkedDAG_MissingChild(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2)},
			{Node: vid(3)},
		},
	}
	nodes := root.TreeNodes(m)
	nodes = nodes[:len(nodes)-1]

	if _, err := BuildLinkedDAG(m, nodes); err == nil {
		t.Fatalf("expected error for missing child node")
	}
}

// TestBuildLinkedDAG_SharedSubtree constructs an input where two parents
// reference an identical (and therefore identically-hashed) subtree. The
// resulting DAG must share a single *LinkedTreeNode for that subtree.
func TestBuildLinkedDAG_SharedSubtree(t *testing.T) {
	m := libxdr.NewXdrMarshaler()

	// Build two identical subtrees: a parent vid(2) with leaf vid(9).
	mkSub := func() *LinkedDAGNode {
		return &LinkedDAGNode{
			Node: vid(2),
			Children: []*LinkedDAGNode{
				{Node: vid(9)},
			},
		}
	}
	subA := mkSub()
	subB := mkSub()

	root := &LinkedDAGNode{
		Node:     vid(1),
		Children: []*LinkedDAGNode{subA, subB},
	}

	// As a tree, the encoding contains duplicate (identical) nodes.
	// BuildLinkedDAG must accept them and dedup.
	nodes := root.TreeNodes(m)

	got, err := BuildLinkedDAG(m, nodes)
	if err != nil {
		t.Fatalf("BuildLinkedDAG: %v", err)
	}

	if len(got.Children) != 2 {
		t.Fatalf("want 2 children at root, got %d", len(got.Children))
	}
	if got.Children[0] != got.Children[1] {
		t.Fatalf("expected shared subtree under both root children")
	}
	if len(got.Children[0].Children) != 1 ||
		got.Children[0].Children[0].Node != vid(9) {
		t.Fatalf("shared subtree structure mismatch")
	}

	// The DAG is structurally indistinguishable from the original tree
	// (same node sequence, same arity), so assertTreeEqual still passes.
	assertNodesEqual(t, root, got)
}

// TestBuildLinkedDAG_DuplicateInputTolerated verifies that BuildLinkedDAG
// does not reject duplicate entries since they encode the same node.
func TestBuildLinkedDAG_DuplicateInputTolerated(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	root := &LinkedDAGNode{
		Node: vid(1),
		Children: []*LinkedDAGNode{
			{Node: vid(2)},
		},
	}
	nodes := root.TreeNodes(m)
	nodes = append(nodes, nodes[1])

	got, err := BuildLinkedDAG(m, nodes)
	if err != nil {
		t.Fatalf("BuildLinkedDAG: %v", err)
	}
	assertNodesEqual(t, root, got)
}
