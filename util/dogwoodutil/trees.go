package dogwoodutil

import (
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type LinkedDAGNode struct {
	Node     proto.ValidatorID
	Children []*LinkedDAGNode
}

func (l *LinkedDAGNode) TreeNodes(m libxdr.XdrMarshaler) proto.DogwoodTreeNodes {
	nodes := make(proto.DogwoodTreeNodes, 1, 1+len(l.Children))
	nodes[0] = proto.DogwoodTreeNode{
		Node:     l.Node,
		Children: make([]proto.DogwoodTreeHash, len(l.Children)),
	}
	for i, ch := range l.Children {
		chNodes := ch.TreeNodes(m)
		nodes[0].Children[i] = TreeNodeHash(m, chNodes[0])
		nodes = append(nodes, chNodes...)
	}
	return nodes
}

// IsTree reports whether the DAG rooted at l is a tree, i.e. no node is
// shared by multiple parents. Since shared subDAGs are represented as shared
// *LinkedDAGNode pointers, this is equivalent to checking that every reachable
// node is visited exactly once.
func (l *LinkedDAGNode) IsTree() bool {
	if l == nil {
		return true
	}
	seen := map[*LinkedDAGNode]struct{}{}
	stack := []*LinkedDAGNode{l}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := seen[n]; ok {
			return false
		}
		seen[n] = struct{}{}
		for _, ch := range n.Children {
			stack = append(stack, ch)
		}
	}
	return true
}

func (l *LinkedDAGNode) FindChildren(v proto.ValidatorID) ([]proto.ValidatorID, bool) {

	if protoutil.ValidatorIDEqual(v, l.Node) {
		children := make([]proto.ValidatorID, len(l.Children))
		for i := range l.Children {
			children[i] = l.Children[i].Node
		}
		return children, true
	}

	for _, ch := range l.Children {
		x, ok := ch.FindChildren(v)
		if ok {
			return x, true
		}
	}

	return nil, false
}

func TreeHash(m libxdr.XdrMarshaler, t *LinkedDAGNode) proto.DogwoodTreeHash {
	return TreeNodeHash(m, t.TreeNodes(m)[0])
}

func TreeNodeHash(m libxdr.XdrMarshaler, treeNode proto.DogwoodTreeNode) proto.DogwoodTreeHash {
	bb := m.MarshalXdr(&treeNode)
	defer bb.Recycle()
	return sha256.Sum256(bb.Bytes())
}

func ReconstructLinkedTree(
	m libxdr.XdrMarshaler,
	h proto.DogwoodTreeHash,
	treeNodes proto.DogwoodTreeNodes,
) (*LinkedDAGNode, error) {

	if len(treeNodes) == 0 {
		return nil, fmt.Errorf("missing tree nodes")
	}
	if TreeNodeHash(m, treeNodes[0]) != h {
		return nil, fmt.Errorf("root tree node hash mismatch")
	}

	return BuildLinkedTree(m, treeNodes)
}

func BuildLinkedDagExpectTree(m libxdr.XdrMarshaler, treeNodes proto.DogwoodTreeNodes) (*LinkedDAGNode, error) {
	t, err := BuildLinkedDAG(m, treeNodes)
	if err != nil {
		return nil, err
	}
	if !t.IsTree() {
		return nil, fmt.Errorf("non-tree route")
	}
	return t, nil
}

// The first node must be the root. All nodes must be part of the same tree.
func BuildLinkedTree(m libxdr.XdrMarshaler, treeNodes proto.DogwoodTreeNodes) (*LinkedDAGNode, error) {

	if len(treeNodes) == 0 {
		return nil, fmt.Errorf("no tree nodes")
	}
	rootHash := TreeNodeHash(m, treeNodes[0])

	// index all nodes by hash
	hs := map[proto.DogwoodTreeHash]proto.DogwoodTreeNode{}
	for _, treeNode := range treeNodes {
		h := TreeNodeHash(m, treeNode)
		if _, ok := hs[h]; ok {
			return nil, fmt.Errorf("duplicate tree node")
		}
		hs[h] = treeNode
	}

	return buildLinkedTree(rootHash, hs)
}

func buildLinkedTree(
	root proto.DogwoodTreeHash,
	hs map[proto.DogwoodTreeHash]proto.DogwoodTreeNode,
) (*LinkedDAGNode, error) {

	makeLinked := func(h proto.DogwoodTreeHash) (*LinkedDAGNode, []proto.DogwoodTreeHash, error) {
		node, ok := hs[h]
		if !ok {
			return nil, nil, fmt.Errorf("missing tree node for hash %x", h)
		}
		return &LinkedDAGNode{Node: node.Node}, node.Children, nil
	}

	type frame struct {
		linked   *LinkedDAGNode
		children []proto.DogwoodTreeHash
	}

	rootLinked, rootChildren, err := makeLinked(root)
	if err != nil {
		return nil, err
	}
	stack := []frame{{linked: rootLinked, children: rootChildren}}

	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		f.linked.Children = make([]*LinkedDAGNode, len(f.children))
		for i, ch := range f.children {
			child, grandChildren, err := makeLinked(ch)
			if err != nil {
				return nil, err
			}
			f.linked.Children[i] = child
			stack = append(stack, frame{linked: child, children: grandChildren})
		}
	}

	return rootLinked, nil
}
