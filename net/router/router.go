package router

import (
	"context"

	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	proto "github.com/groundhogchain/protocol-go"
)

type ChannelID = proto.BroadcastChannelID
type ValidatorID = proto.ValidatorID
type MulticastTreeNode = proto.MulticastTreeNode
type MulticastTreeNodes = proto.MulticastTreeNodes
type ValidatorPublicInfo = proto.ValidatorPublicInfo
type MulticastTreeHash = proto.MulticastTreeHash

type Router interface {
	Self() ValidatorPublicInfo
	BuildTree(ValidatorID, ChannelID) (MulticastTreeNodes, []ValidatorPublicInfo, MulticastTreeHash)
	ChooseCatchupProviders(ctx context.Context, seed int64) []ValidatorPublicInfo
	DogwoodRouter
}

type DogwoodRouter interface {
	BuildDogwoodTree(ValidatorID) *dogwoodutil.LinkedTreeNode
	ResolveAddress(ValidatorID) (string, error)
}
