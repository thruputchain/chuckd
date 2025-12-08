package dogwoodutil

import (
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type LinkedTreeNode struct {
	Node     proto.ValidatorID
	Children []*LinkedTreeNode
}

func (l *LinkedTreeNode) TreeNodes(m libxdr.XdrMarshaler) proto.DogwoodTreeNodes {
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

func (l *LinkedTreeNode) FindChildren(v proto.ValidatorID) ([]proto.ValidatorID, bool) {

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

func TreeHash(m libxdr.XdrMarshaler, t *LinkedTreeNode) proto.DogwoodTreeHash {
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
) (*LinkedTreeNode, error) {

	if len(treeNodes) == 0 {
		return nil, fmt.Errorf("missing tree nodes")
	}
	if TreeNodeHash(m, treeNodes[0]) != h {
		return nil, fmt.Errorf("root tree node hash mismatch")
	}

	return BuildLinkedTree(m, treeNodes)
}

// The first node must be the root. All nodes must be part of the same tree.
func BuildLinkedTree(m libxdr.XdrMarshaler, treeNodes proto.DogwoodTreeNodes) (*LinkedTreeNode, error) {

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
) (*LinkedTreeNode, error) {

	node, ok := hs[root]
	if !ok {
		return nil, fmt.Errorf("missing root node")
	}

	linked := &LinkedTreeNode{
		Node: node.Node,
	}
	for _, ch := range node.Children {
		l, err := buildLinkedTree(ch, hs)
		if err != nil {
			return nil, err
		}
		linked.Children = append(linked.Children, l)
	}

	return linked, nil
}
