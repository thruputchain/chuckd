package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func MulticastTreeNodeHash(m xdrutil.XdrMarshaler, treeNode proto.MulticastTreeNode) proto.MulticastTreeHash {
	bb := m.MarshalXdr(&treeNode)
	defer bb.Recycle()
	return sha256.Sum256(bb.Bytes())
}

type LinkedTreeNode struct {
	Node     proto.ValidatorPublicInfo
	Children []*LinkedTreeNode
	Source   proto.MulticastTreeNode
}

func (l *LinkedTreeNode) FindChildren(v proto.ValidatorPublicInfo) ([]proto.ValidatorPublicInfo, bool) {

	if ValidatorIDEqual(
		PublicVID(v),
		PublicVID(l.Node),
	) {
		children := make([]proto.ValidatorPublicInfo, len(l.Children))
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

func VerifyBuildLinkedTree(m xdrutil.XdrMarshaler, h proto.MulticastTreeHash, treeNodes proto.MulticastTreeNodes) *LinkedTreeNode {

	if len(treeNodes) == 0 {
		return nil
	}
	if MulticastTreeNodeHash(m, treeNodes[0]) != h {
		return nil
	}

	return BuildLinkedTree(m, treeNodes)
}

// BuildLinkedTree reconstructs a linked tree from the Merkle MulticastTreeNodes.
// The first node must be the root. All nodes must be part of the same tree.
func BuildLinkedTree(m xdrutil.XdrMarshaler, treeNodes proto.MulticastTreeNodes) *LinkedTreeNode {

	if len(treeNodes) == 0 {
		return nil
	}
	rootHash := MulticastTreeNodeHash(m, treeNodes[0])

	// index all nodes by hash
	hs := map[proto.MulticastTreeHash]proto.MulticastTreeNode{}
	for _, treeNode := range treeNodes {
		h := MulticastTreeNodeHash(m, treeNode)
		if _, ok := hs[h]; ok {
			return nil // duplicate
		}
		hs[h] = treeNode
	}

	return buildLinkedTree(rootHash, hs)
}

func buildLinkedTree(
	root proto.MulticastTreeHash,
	hs map[proto.MulticastTreeHash]proto.MulticastTreeNode,
) *LinkedTreeNode {

	node, ok := hs[root]
	if !ok {
		return nil
	}

	linked := &LinkedTreeNode{
		Node:   node.Node,
		Source: node,
	}
	for _, ch := range node.Children {
		l := buildLinkedTree(ch, hs)
		if l == nil {
			return nil
		}
		linked.Children = append(linked.Children, l)
	}

	return linked
}
