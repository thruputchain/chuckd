package router

import (
	"context"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestRouter struct {
	Trees map[TestTreeKey]*TestTreeNode
}

type TestTreeKey struct {
	VID ValidatorID
}

type TestTreeNode struct {
	V        ValidatorPublicInfo
	Children []*TestTreeNode
}

func TestTreeMerkle(tt *TestTreeNode) MulticastTreeNodes {
	node := MulticastTreeNode{Node: tt.V}
	all := MulticastTreeNodes{}
	for _, c := range tt.Children {
		subNodes := TestTreeMerkle(c)
		all = append(all, subNodes...)
		node.Children = append(node.Children, protoutil.MulticastTreeNodeHash(subNodes[0]))
	}
	return append(MulticastTreeNodes{node}, all...)
}

func (x *TestRouter) Self() proto.ValidatorPublicInfo {
	panic("not used")
}

func (x *TestRouter) ChooseCatchupProviders(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	panic("not used")
}

func (x *TestRouter) BuildTree(vid ValidatorID, _ ChannelID) (MulticastTreeNodes, []ValidatorPublicInfo, MulticastTreeHash) {
	tt := x.Trees[TestTreeKey{VID: vid}]
	nodes := TestTreeMerkle(tt)
	children := make([]ValidatorPublicInfo, len(tt.Children))
	for i := range tt.Children {
		children[i] = tt.Children[i].V
	}
	return nodes, children, protoutil.MulticastTreeNodeHash(nodes[0])
}
