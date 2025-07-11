package router

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

var testM = xdrutil.NewXdrMarshaler()

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
		node.Children = append(node.Children, protoutil.MulticastTreeNodeHash(testM, subNodes[0]))
	}
	return append(MulticastTreeNodes{node}, all...)
}

func (x *TestRouter) Self() proto.ValidatorPublicInfo {
	panic("not used")
}

func (x *TestRouter) ResolveAddress(vid proto.ValidatorID) (string, error) {
	n, ok := x.Trees[TestTreeKey{vid}]
	if !ok {
		return "", fmt.Errorf("validator not found")
	}
	return n.V.Address, nil
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
	return nodes, children, protoutil.MulticastTreeNodeHash(testM, nodes[0])
}

func (x *TestRouter) BuildDogwoodTree(vid ValidatorID) *dogwoodutil.LinkedTreeNode {
	return TestTreeLinked(x.Trees[TestTreeKey{VID: vid}])
}

func TestTreeLinked(tt *TestTreeNode) *dogwoodutil.LinkedTreeNode {
	l := &dogwoodutil.LinkedTreeNode{
		Node:     protoutil.VIDOfPublic(tt.V),
		Children: make([]*dogwoodutil.LinkedTreeNode, len(tt.Children)),
	}
	for i, tc := range tt.Children {
		l.Children[i] = TestTreeLinked(tc)
	}
	return l
}
