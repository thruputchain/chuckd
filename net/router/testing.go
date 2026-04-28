package router

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

var testM = libxdr.NewXdrMarshaler()

type TestRouter struct {
	self  proto.ValidatorPublicInfo
	trees map[TestTreeKey]*TestTreeNode
}

type TestTreeKey struct {
	VID ValidatorID
}

type TestTreeNode struct {
	V        ValidatorPublicInfo
	Children []*TestTreeNode
}

func NewTestRouter(self proto.ValidatorPublicInfo, trees map[TestTreeKey]*TestTreeNode) *TestRouter {
	return &TestRouter{
		self:  self,
		trees: trees,
	}
}

func (x *TestRouter) Self() proto.ValidatorPublicInfo {
	return x.self
}

func (x *TestRouter) OurQuorumSet() proto.SCPQuorumSet {
	panic("not used here")
}

func (x *TestRouter) ResolveAddress(vid proto.ValidatorID) (string, error) {
	n, ok := x.trees[TestTreeKey{vid}]
	if !ok {
		return "", fmt.Errorf("validator not found")
	}
	return n.V.Address, nil
}

func (x *TestRouter) ChooseSCPProviders(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	panic("not used")
}

func (x *TestRouter) ChooseCatchupProviders(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	panic("not used")
}

func (x *TestRouter) BuildDogwoodTree(vid ValidatorID, maxSize int) *dogwoodutil.LinkedDAGNode {
	return TestTreeLinked(x.trees[TestTreeKey{VID: vid}])
}

func TestTreeLinked(tt *TestTreeNode) *dogwoodutil.LinkedDAGNode {
	l := &dogwoodutil.LinkedDAGNode{
		Node:     protoutil.PublicVID(tt.V),
		Children: make([]*dogwoodutil.LinkedDAGNode, len(tt.Children)),
	}
	for i, tc := range tt.Children {
		l.Children[i] = TestTreeLinked(tc)
	}
	return l
}
