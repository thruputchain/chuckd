package router

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type FixedRouter struct {
	maxFanOut  int
	self       proto.ValidatorPublicInfo
	validators []proto.ValidatorPublicInfo
}

func NewFixedRouter(
	ctx context.Context,
	self proto.ValidatorPublicInfo,
	vs []proto.ValidatorPublicInfo,
	maxFanOut int,
) Router {
	return &FixedRouter{
		maxFanOut:  maxFanOut,
		self:       self,
		validators: vs,
	}
}

func (x *FixedRouter) BuildTree(rootID ValidatorID, cid ChannelID) (
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
	treeHash MulticastTreeHash,
) {
	vs := append([]ValidatorPublicInfo{x.self}, x.validators...)
	return buildTree(x.maxFanOut, vs, rootID, cid)
}
