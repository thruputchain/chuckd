package dogwoodutil

import (
	"crypto/sha256"
	"fmt"

	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
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

	return BuildLinkedDagExpectTree(m, treeNodes)
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
